package extension

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"extension-scaffold/internal/config"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
)

// ChainClient is the slice of ethclient.Client the watcher needs; mocked in
// unit tests, satisfied by *ethclient.Client on the real chain.
type ChainClient interface {
	CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error)
	SendTransaction(ctx context.Context, tx *ethtypes.Transaction) error
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*ethtypes.Receipt, error)
	TransactionByHash(ctx context.Context, txHash common.Hash) (*ethtypes.Transaction, bool, error)
	ChainID(ctx context.Context) (*big.Int, error)
}

const (
	// settleMaxAgeSec is the FTSO price staleness bound passed to
	// DarkStopVault.settle (which re-checks it on-chain).
	settleMaxAgeSec = 300
	// payoutDecimals matches DarkStopVault.PAYOUT_DECIMALS: trigger prices
	// and normalized feed values are USD/FLR at 6 decimals.
	payoutDecimals = 6
	// settleAttempts is the number of send tries per triggered order per tick.
	settleAttempts = 3
	// This many consecutive receipt + transaction lookup misses classify a
	// broadcast as dropped. A known mempool transaction is fee-bumped after
	// pendingReplacementAge so it cannot block an order forever.
	pendingDropChecks     = 3
	pendingReplacementAge = 30 * time.Second
)

// flrUsdFeedID is the FTSO v2 FLR/USD feed id (bytes21, matches vault const).
var flrUsdFeedID = [21]byte{0x01, 'F', 'L', 'R', '/', 'U', 'S', 'D'}

// watcherABI is a minimal hand-rolled binding: the vault's FTSO_V2() and
// settle(), plus FtsoV2's getFeedById(). Generated bindings are overkill for
// three methods.
var watcherABI = func() abi.ABI {
	const abiJSON = `[
		{"name":"FTSO_V2","type":"function","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
		{"name":"orders","type":"function","stateMutability":"view","inputs":[{"name":"","type":"uint256"}],"outputs":[{"name":"owner","type":"address"},{"name":"deposit","type":"uint256"},{"name":"status","type":"uint8"}]},
		{"name":"settle","type":"function","stateMutability":"nonpayable","inputs":[{"name":"_orderId","type":"uint256"},{"name":"_triggerPrice","type":"uint256"},{"name":"_maxAgeSec","type":"uint256"}],"outputs":[]},
		{"name":"getFeedById","type":"function","stateMutability":"view","inputs":[{"name":"_feedId","type":"bytes21"}],"outputs":[{"name":"","type":"uint256"},{"name":"","type":"int8"},{"name":"","type":"uint64"}]}
	]`
	parsed, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		panic(fmt.Sprintf("parsing watcher ABI: %v", err))
	}
	return parsed
}()

// Watcher polls the FTSO FLR/USD feed and settles open orders whose trigger
// price has been hit. Every step is logged: that log is the audit trail of
// what the enclave did and when.
type Watcher struct {
	client   ChainClient
	store    *Store
	vault    common.Address
	key      *ecdsa.PrivateKey
	from     common.Address
	interval time.Duration

	// sleep is time.Sleep, injectable in tests.
	sleep func(time.Duration)

	// lazily resolved / cached chain facts.
	ftso    common.Address
	chainID *big.Int

	// pending tracks every broadcast until chain state is reconciled. Receipt
	// checks are non-blocking, so price sampling never pauses behind mining.
	pending map[uint64]pendingSettlement

	// tickMu serializes all watcher state. nonceOwners prevents two orders
	// from receiving the same executor nonce when an RPC's pending nonce lags
	// behind broadcasts made earlier in the same tick.
	tickMu      sync.Mutex
	nonceOwners map[uint64]uint64 // nonce -> order ID
	nextNonce   uint64
	nonceReady  bool
}

type pendingSettlement struct {
	txHash        common.Hash
	nonce         uint64
	gasPrice      *big.Int
	triggerPrice  *big.Int
	firstSeen     time.Time
	missingChecks int
}

// NewWatcher builds a watcher settling via the given executor key
// (hex, with or without 0x prefix — must match vault.teeExecutor).
func NewWatcher(client ChainClient, store *Store, vault common.Address, executorKeyHex string, interval time.Duration) (*Watcher, error) {
	key, err := ethcrypto.HexToECDSA(strings.TrimPrefix(executorKeyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("parsing executor private key: %w", err)
	}
	return &Watcher{
		client:      client,
		store:       store,
		vault:       vault,
		key:         key,
		from:        ethcrypto.PubkeyToAddress(key.PublicKey),
		interval:    interval,
		sleep:       time.Sleep,
		pending:     make(map[uint64]pendingSettlement),
		nonceOwners: make(map[uint64]uint64),
	}, nil
}

// Run polls until the context is cancelled.
func (w *Watcher) Run(ctx context.Context) {
	logger.Infof("watcher: started — vault %s, executor %s, polling every %s", w.vault, w.from, w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("watcher: stopped")
			return
		case <-ticker.C:
			if err := w.tick(ctx); err != nil {
				logger.Warnf("watcher: tick failed: %v", err)
			}
		}
	}
}

// tick reads the FTSO price once and settles every open order whose trigger
// is at or above the current price.
func (w *Watcher) tick(ctx context.Context) error {
	w.tickMu.Lock()
	defer w.tickMu.Unlock()

	price, ts, err := w.currentPrice(ctx)
	if err != nil {
		w.reconcilePending(ctx, nil)
		return fmt.Errorf("reading FTSO price: %w", err)
	}

	age := time.Now().Unix() - int64(ts) // #nosec G115 -- feed timestamps fit int64 for eons
	if age < 0 {
		w.reconcilePending(ctx, nil)
		return fmt.Errorf("FTSO feed timestamp is %ds in the future", -age)
	}
	if age > settleMaxAgeSec {
		w.reconcilePending(ctx, nil)
		return fmt.Errorf("FTSO feed is stale: %ds old (max %ds)", age, settleMaxAgeSec)
	}

	open := w.store.OpenOrders()
	logger.Infof("watcher: FLR/USD = %s (6dp, feed age %ds), %d open order(s)", price, age, len(open))
	w.store.ObservePrice(price, time.Unix(int64(ts), 0)) // #nosec G115 -- validated above
	triggered := w.store.TriggeredOrders(price)

	// Sample and update every private high-watermark before receipt RPC work.
	// This keeps trailing semantics responsive even with many pending txs.
	w.reconcilePending(ctx, price)

	for _, order := range triggered {
		// One executor transaction at a time prevents nonce gaps. Ethereum
		// cannot mine nonce n+1 while n is absent; serial submission makes a
		// dropped inactive order unable to strand later settlements.
		if len(w.pending) > 0 {
			break
		}
		current, exists := w.store.Get(order.ID)
		if !exists || current.Status != StatusOpen {
			continue
		}
		logger.Infof("watcher: order %d TRIGGERED (price %s <= trigger %s) — settling", order.ID, price, order.TriggerPrice)
		settled, pending := w.settleWithRetry(ctx, order)
		if settled {
			if w.store.MarkExecuted(order.ID) {
				logger.Infof("watcher: order %d marked executed", order.ID)
			} else {
				logger.Warnf("watcher: order %d confirmed but local state was no longer open", order.ID)
			}
		} else if !pending {
			logger.Warnf("watcher: order %d settle failed after %d attempts, will retry next tick", order.ID, settleAttempts)
		}
	}
	return nil
}

type unconfirmedTxState uint8

const (
	txStuck unconfirmedTxState = iota
	txDropped
	txMinedNoReceipt
)

// reconcilePending performs one non-blocking receipt lookup per broadcast. A
// successful receipt repairs local state; a reverted receipt releases the
// order; a stale mempool transaction is safely fee-bumped with the same nonce.
// sampledPrice is nil when the tick has no fresh FTSO sample; replacement is
// then deferred because the watcher cannot prove the order is still active.
func (w *Watcher) reconcilePending(ctx context.Context, sampledPrice *big.Int) {
	for orderID, pending := range w.pending {
		receipt, err := w.client.TransactionReceipt(ctx, pending.txHash)
		if errors.Is(err, ethereum.NotFound) {
			w.reconcileUnconfirmed(ctx, orderID, pending, sampledPrice)
			continue
		}
		if err != nil {
			logger.Warnf("watcher: order %d pending tx %s receipt check failed: %v", orderID, pending.txHash, err)
			continue
		}

		delete(w.pending, orderID)
		w.releaseNonce(orderID, pending.nonce, true)
		if receipt.Status == ethtypes.ReceiptStatusSuccessful {
			if w.store.MarkExecuted(orderID) {
				logger.Infof("watcher: order %d pending tx %s confirmed — local state reconciled", orderID, pending.txHash)
			} else {
				logger.Warnf("watcher: order %d pending tx %s confirmed but local state was no longer open", orderID, pending.txHash)
			}
			continue
		}
		logger.Warnf("watcher: order %d pending tx %s reverted — releasing for retry", orderID, pending.txHash)
	}
}

func (w *Watcher) reconcileUnconfirmed(ctx context.Context, orderID uint64, pending pendingSettlement, sampledPrice *big.Int) {
	_, isPending, err := w.client.TransactionByHash(ctx, pending.txHash)
	if err == nil {
		if isPending {
			pending.missingChecks = 0
			w.pending[orderID] = pending
			if time.Since(pending.firstSeen) < pendingReplacementAge {
				return
			}
			w.reconcileAgainstVault(ctx, orderID, pending, sampledPrice, txStuck, "stuck in mempool")
			return
		}
		// The nonce is already consumed. A lagging receipt must never cause a
		// same-nonce replacement; canonical vault state decides local repair.
		w.reconcileAgainstVault(ctx, orderID, pending, sampledPrice, txMinedNoReceipt, "mined without receipt")
		return
	}
	if err != nil && !errors.Is(err, ethereum.NotFound) {
		logger.Warnf("watcher: order %d pending tx %s lookup failed: %v", orderID, pending.txHash, err)
		return
	}

	pending.missingChecks++
	w.pending[orderID] = pending
	if pending.missingChecks < pendingDropChecks {
		return
	}
	w.reconcileAgainstVault(ctx, orderID, pending, sampledPrice, txDropped, "dropped from mempool")
}

func (w *Watcher) reconcileAgainstVault(
	ctx context.Context,
	orderID uint64,
	pending pendingSettlement,
	sampledPrice *big.Int,
	txState unconfirmedTxState,
	reason string,
) {
	status, err := w.onChainOrderStatus(ctx, orderID)
	if err != nil {
		logger.Warnf("watcher: order %d could not reconcile tx %s against vault: %v", orderID, pending.txHash, err)
		return
	}
	switch status {
	case 2: // DarkStopVault.STATUS_EXECUTED
		delete(w.pending, orderID)
		w.releaseNonce(orderID, pending.nonce, txState == txMinedNoReceipt)
		if w.store.MarkExecuted(orderID) {
			logger.Infof("watcher: order %d is executed on-chain — local state reconciled without receipt", orderID)
		}
	case 1: // DarkStopVault.STATUS_OPEN
		if txState == txMinedNoReceipt {
			delete(w.pending, orderID)
			w.releaseNonce(orderID, pending.nonce, true)
			logger.Warnf("watcher: order %d tx %s was mined without execution — releasing consumed nonce for a fresh settlement", orderID, pending.txHash)
			return
		}

		currentTrigger, triggerOK := w.store.EffectiveTrigger(orderID)
		active := sampledPrice != nil && triggerOK && sampledPrice.Cmp(currentTrigger) <= 0
		if !active {
			if txState == txDropped && sampledPrice != nil {
				delete(w.pending, orderID)
				w.releaseNonce(orderID, pending.nonce, false)
				logger.Infof("watcher: order %d tx %s dropped while policy is not currently triggered — waiting for a fresh crossing", orderID, pending.txHash)
			}
			return
		}
		if err := w.replacePending(ctx, orderID, pending, currentTrigger); err != nil {
			logger.Warnf("watcher: order %d tx %s %s; fee-bump failed: %v", orderID, pending.txHash, reason, err)
		}
	case 0, 3: // unknown or cancelled
		delete(w.pending, orderID)
		w.releaseNonce(orderID, pending.nonce, txState == txMinedNoReceipt)
		w.store.Delete(orderID)
		logger.Warnf("watcher: order %d is no longer open on-chain (status %d) — removed locally", orderID, status)
	default:
		logger.Warnf("watcher: order %d has unknown on-chain status %d; retaining pending tx", orderID, status)
	}
}

func (w *Watcher) replacePending(ctx context.Context, orderID uint64, pending pendingSettlement, currentTrigger *big.Int) error {
	if owner, ok := w.nonceOwners[pending.nonce]; ok && owner != orderID {
		return fmt.Errorf("nonce %d is owned by order %d", pending.nonce, owner)
	}
	w.nonceOwners[pending.nonce] = orderID
	data, err := packSettleCalldata(orderID, currentTrigger, settleMaxAgeSec)
	if err != nil {
		return fmt.Errorf("packing replacement calldata: %w", err)
	}
	suggested, err := w.client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("fetching replacement gas price: %w", err)
	}
	bumped := new(big.Int).Mul(pending.gasPrice, big.NewInt(9))
	bumped.Div(bumped, big.NewInt(8))
	bumped.Add(bumped, big.NewInt(1))
	if suggested.Cmp(bumped) > 0 {
		bumped = new(big.Int).Set(suggested)
	}
	gas, err := w.client.EstimateGas(ctx, ethereum.CallMsg{From: w.from, To: &w.vault, Data: data})
	if err != nil {
		return fmt.Errorf("estimating replacement gas: %w", err)
	}
	if err := w.ensureChainID(ctx); err != nil {
		return err
	}
	tx := ethtypes.NewTransaction(pending.nonce, w.vault, big.NewInt(0), gas, bumped, data)
	signed, err := ethtypes.SignTx(tx, ethtypes.LatestSignerForChainID(w.chainID), w.key)
	if err != nil {
		return fmt.Errorf("signing replacement tx: %w", err)
	}
	if err := w.client.SendTransaction(ctx, signed); err != nil {
		return fmt.Errorf("sending replacement tx: %w", err)
	}
	pending.txHash = signed.Hash()
	pending.gasPrice = new(big.Int).Set(bumped)
	pending.triggerPrice = new(big.Int).Set(currentTrigger)
	pending.firstSeen = time.Now()
	pending.missingChecks = 0
	w.pending[orderID] = pending
	logger.Infof("watcher: order %d settlement fee-bumped: %s (nonce %d, gasPrice %s)", orderID, signed.Hash(), pending.nonce, bumped)
	return nil
}

func (w *Watcher) onChainOrderStatus(ctx context.Context, orderID uint64) (uint8, error) {
	input, err := watcherABI.Pack("orders", new(big.Int).SetUint64(orderID))
	if err != nil {
		return 0, fmt.Errorf("packing orders lookup: %w", err)
	}
	output, err := w.client.CallContract(ctx, ethereum.CallMsg{To: &w.vault, Data: input}, nil)
	if err != nil {
		return 0, fmt.Errorf("calling orders lookup: %w", err)
	}
	values, err := watcherABI.Unpack("orders", output)
	if err != nil {
		return 0, fmt.Errorf("unpacking orders lookup: %w", err)
	}
	status, ok := values[2].(uint8)
	if !ok {
		return 0, fmt.Errorf("order status: expected uint8, got %T", values[2])
	}
	return status, nil
}

// currentPrice returns the FLR/USD price normalized to 6 decimals plus the
// feed timestamp.
func (w *Watcher) currentPrice(ctx context.Context) (*big.Int, uint64, error) {
	ftso, err := w.ftsoAddress(ctx)
	if err != nil {
		return nil, 0, err
	}

	input, err := watcherABI.Pack("getFeedById", flrUsdFeedID)
	if err != nil {
		return nil, 0, fmt.Errorf("packing getFeedById: %w", err)
	}
	output, err := w.client.CallContract(ctx, ethereum.CallMsg{To: &ftso, Data: input}, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("calling getFeedById: %w", err)
	}
	values, err := watcherABI.Unpack("getFeedById", output)
	if err != nil {
		return nil, 0, fmt.Errorf("unpacking getFeedById: %w", err)
	}
	value, ok := values[0].(*big.Int)
	if !ok {
		return nil, 0, fmt.Errorf("feed value: expected *big.Int, got %T", values[0])
	}
	decimals, ok := values[1].(int8)
	if !ok {
		return nil, 0, fmt.Errorf("feed decimals: expected int8, got %T", values[1])
	}
	ts, ok := values[2].(uint64)
	if !ok {
		return nil, 0, fmt.Errorf("feed timestamp: expected uint64, got %T", values[2])
	}

	return normalizePrice(value, decimals), ts, nil
}

// ftsoAddress resolves (once) the FtsoV2 address from the vault's immutable.
func (w *Watcher) ftsoAddress(ctx context.Context) (common.Address, error) {
	if w.ftso != (common.Address{}) {
		return w.ftso, nil
	}
	input, err := watcherABI.Pack("FTSO_V2")
	if err != nil {
		return common.Address{}, fmt.Errorf("packing FTSO_V2: %w", err)
	}
	output, err := w.client.CallContract(ctx, ethereum.CallMsg{To: &w.vault, Data: input}, nil)
	if err != nil {
		return common.Address{}, fmt.Errorf("calling vault.FTSO_V2: %w", err)
	}
	values, err := watcherABI.Unpack("FTSO_V2", output)
	if err != nil {
		return common.Address{}, fmt.Errorf("unpacking FTSO_V2: %w", err)
	}
	addr, ok := values[0].(common.Address)
	if !ok {
		return common.Address{}, fmt.Errorf("FTSO_V2: expected address, got %T", values[0])
	}
	w.ftso = addr
	logger.Infof("watcher: resolved FtsoV2 at %s from vault", addr)
	return addr, nil
}

// normalizePrice rescales a feed value with int8 decimals to payoutDecimals,
// mirroring DarkStopVault._toPayoutDecimals.
func normalizePrice(value *big.Int, decimals int8) *big.Int {
	shift := payoutDecimals - int(decimals)
	if shift >= 0 {
		mul := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(shift)), nil)
		return new(big.Int).Mul(value, mul)
	}
	div := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-shift)), nil)
	return new(big.Int).Div(value, div)
}

// packSettleCalldata encodes settle(orderId, triggerPrice, maxAgeSec).
func packSettleCalldata(orderID uint64, triggerPrice *big.Int, maxAgeSec uint64) ([]byte, error) {
	return watcherABI.Pack("settle",
		new(big.Int).SetUint64(orderID),
		triggerPrice,
		new(big.Int).SetUint64(maxAgeSec),
	)
}

// reserveNonce gives an order exclusive ownership of one executor nonce.
// The local floor advances after each reservation so multiple broadcasts in
// one tick remain unique even when PendingNonceAt is served by a lagging RPC.
// A retry for the same order deliberately reuses its reservation.
func (w *Watcher) reserveNonce(ctx context.Context, orderID uint64) (uint64, error) {
	for nonce, owner := range w.nonceOwners {
		if owner == orderID {
			return nonce, nil
		}
	}

	remote, err := w.client.PendingNonceAt(ctx, w.from)
	if err != nil {
		return 0, fmt.Errorf("fetching nonce: %w", err)
	}
	candidate := remote
	if w.nonceReady && w.nextNonce > candidate {
		candidate = w.nextNonce
	}
	for {
		if _, occupied := w.nonceOwners[candidate]; !occupied {
			break
		}
		if candidate == ^uint64(0) {
			return 0, fmt.Errorf("executor nonce space exhausted")
		}
		candidate++
	}
	if candidate == ^uint64(0) {
		return 0, fmt.Errorf("executor nonce space exhausted")
	}
	w.nonceOwners[candidate] = orderID
	w.nextNonce = candidate + 1
	w.nonceReady = true
	return candidate, nil
}

// releaseNonce clears ownership after a terminal chain result. A transaction
// proven mined consumed its nonce. A transaction proven dropped did not, so
// the local floor may safely return to that gap while ownership checks still
// protect every other in-flight order.
func (w *Watcher) releaseNonce(orderID, nonce uint64, consumed bool) {
	owner, ok := w.nonceOwners[nonce]
	if !ok || owner != orderID {
		return
	}
	delete(w.nonceOwners, nonce)
	if !consumed && (!w.nonceReady || nonce < w.nextNonce) {
		w.nextNonce = nonce
		w.nonceReady = true
	}
}

func (w *Watcher) releaseOrderNonce(orderID uint64, consumed bool) {
	for nonce, owner := range w.nonceOwners {
		if owner == orderID {
			w.releaseNonce(orderID, nonce, consumed)
			return
		}
	}
}

// settleWithRetry tries to settle one order up to settleAttempts times with
// exponential backoff. It returns whether settlement was confirmed and
// whether a broadcast transaction is still awaiting its receipt.
// This reveals the trigger price on-chain — by design: settlement is the
// moment of disclosure, and the vault re-verifies it against the live FTSO.
func (w *Watcher) settleWithRetry(ctx context.Context, order Order) (bool, bool) {
	for attempt := 1; attempt <= settleAttempts; attempt++ {
		logger.Infof("watcher: order %d settle attempt %d/%d", order.ID, attempt, settleAttempts)
		err := w.settleOnce(ctx, order)
		if err == nil {
			return true, false
		}
		var unresolved *pendingTransactionError
		if errors.As(err, &unresolved) {
			w.pending[order.ID] = unresolved.pending
			logger.Infof("watcher: order %d tx %s has an unresolved broadcast/receipt — reconciling on later ticks", order.ID, unresolved.pending.txHash)
			return false, true
		}
		logger.Warnf("watcher: order %d settle attempt %d failed: %v", order.ID, attempt, err)
		if attempt < settleAttempts {
			backoff := time.Second << (attempt - 1) // 1s, 2s
			logger.Infof("watcher: order %d retrying in %s", order.ID, backoff)
			w.sleep(backoff)
		}
	}
	w.releaseOrderNonce(order.ID, false)
	return false, false
}

type pendingTransactionError struct {
	pending pendingSettlement
	err     error
}

func (e *pendingTransactionError) Error() string {
	return fmt.Sprintf("transaction %s awaits reconciliation: %v", e.pending.txHash, e.err)
}

func (e *pendingTransactionError) Unwrap() error { return e.err }

// settleOnce builds, signs and sends one settle transaction. It performs at
// most one immediate receipt lookup; mining is otherwise reconciled on later
// ticks so the watcher can keep sampling FTSO without blocking.
func (w *Watcher) settleOnce(ctx context.Context, order Order) error {
	data, err := packSettleCalldata(order.ID, order.TriggerPrice, settleMaxAgeSec)
	if err != nil {
		return fmt.Errorf("packing settle calldata: %w", err)
	}

	if err := w.ensureChainID(ctx); err != nil {
		return err
	}

	gasPrice, err := w.client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("fetching gas price: %w", err)
	}
	gas, err := w.client.EstimateGas(ctx, ethereum.CallMsg{From: w.from, To: &w.vault, Data: data})
	if err != nil {
		return fmt.Errorf("estimating gas: %w", err)
	}
	nonce, err := w.reserveNonce(ctx, order.ID)
	if err != nil {
		return err
	}

	tx := ethtypes.NewTransaction(nonce, w.vault, big.NewInt(0), gas, gasPrice, data)
	signed, err := ethtypes.SignTx(tx, ethtypes.LatestSignerForChainID(w.chainID), w.key)
	if err != nil {
		w.releaseNonce(order.ID, nonce, false)
		return fmt.Errorf("signing settle tx: %w", err)
	}

	pending := pendingSettlement{
		txHash:       signed.Hash(),
		nonce:        nonce,
		gasPrice:     new(big.Int).Set(gasPrice),
		triggerPrice: new(big.Int).Set(order.TriggerPrice),
		firstSeen:    time.Now(),
	}
	if err := w.client.SendTransaction(ctx, signed); err != nil {
		// A timeout, disconnect, or "already known" response does not prove
		// rejection. Preserve the signed hash and nonce until chain lookups
		// establish whether it is pending, mined, or genuinely dropped.
		return &pendingTransactionError{pending: pending, err: fmt.Errorf("broadcast result uncertain: %w", err)}
	}
	logger.Infof("watcher: order %d settle tx sent: %s (nonce %d, gas %d, gasPrice %s)",
		order.ID, signed.Hash(), nonce, gas, gasPrice)

	receipt, err := w.client.TransactionReceipt(ctx, signed.Hash())
	if err != nil {
		return &pendingTransactionError{pending: pending, err: err}
	}
	w.releaseNonce(order.ID, nonce, true)
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("settle tx %s reverted on-chain", signed.Hash())
	}
	logger.Infof("watcher: order %d settle confirmed in tx %s", order.ID, signed.Hash())
	return nil
}

func (w *Watcher) ensureChainID(ctx context.Context) error {
	if w.chainID != nil {
		return nil
	}
	chainID, err := w.client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("fetching chain id: %w", err)
	}
	w.chainID = chainID
	return nil
}

// LaunchWatcherFromConfig dials CHAIN_URL and starts the watcher goroutine if
// CHAIN_URL, VAULT_ADDRESS and EXECUTOR_PRIVATE_KEY are all configured.
// Returns false (and no error) when the wiring is absent: the extension then
// runs handler-only, which is fine for local decode/decrypt testing.
func LaunchWatcherFromConfig(ctx context.Context, store *Store) (bool, error) {
	if config.ChainURL == "" || config.VaultAddress == "" || config.ExecutorPrivateKey == "" {
		return false, nil
	}
	if !common.IsHexAddress(config.VaultAddress) {
		return false, fmt.Errorf("VAULT_ADDRESS %q is not a valid address", config.VaultAddress)
	}
	client, err := ethclient.DialContext(ctx, config.ChainURL)
	if err != nil {
		return false, fmt.Errorf("dialing CHAIN_URL: %w", err)
	}
	w, err := NewWatcher(client, store, common.HexToAddress(config.VaultAddress), config.ExecutorPrivateKey, config.WatcherInterval)
	if err != nil {
		return false, err
	}
	go w.Run(ctx)
	return true, nil
}
