package extension

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"strings"
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
	// receiptPolls / receiptPollDelay bound how long a sent settle tx is
	// awaited before the attempt is written off.
	receiptPolls     = 30
	receiptPollDelay = 2 * time.Second
)

// flrUsdFeedID is the FTSO v2 FLR/USD feed id (bytes21, matches vault const).
var flrUsdFeedID = [21]byte{0x01, 'F', 'L', 'R', '/', 'U', 'S', 'D'}

// watcherABI is a minimal hand-rolled binding: the vault's FTSO_V2() and
// settle(), plus FtsoV2's getFeedById(). Generated bindings are overkill for
// three methods.
var watcherABI = func() abi.ABI {
	const abiJSON = `[
		{"name":"FTSO_V2","type":"function","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
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
}

// NewWatcher builds a watcher settling via the given executor key
// (hex, with or without 0x prefix — must match vault.teeExecutor).
func NewWatcher(client ChainClient, store *Store, vault common.Address, executorKeyHex string, interval time.Duration) (*Watcher, error) {
	key, err := ethcrypto.HexToECDSA(strings.TrimPrefix(executorKeyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("parsing executor private key: %w", err)
	}
	return &Watcher{
		client:   client,
		store:    store,
		vault:    vault,
		key:      key,
		from:     ethcrypto.PubkeyToAddress(key.PublicKey),
		interval: interval,
		sleep:    time.Sleep,
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
	price, ts, err := w.currentPrice(ctx)
	if err != nil {
		return fmt.Errorf("reading FTSO price: %w", err)
	}

	age := time.Now().Unix() - int64(ts) // #nosec G115 -- feed timestamps fit int64 for eons
	if age > settleMaxAgeSec {
		return fmt.Errorf("FTSO feed is stale: %ds old (max %ds)", age, settleMaxAgeSec)
	}

	open := w.store.OpenOrders()
	logger.Infof("watcher: FLR/USD = %s (6dp, feed age %ds), %d open order(s)", price, age, len(open))

	for _, order := range open {
		if price.Cmp(order.TriggerPrice) > 0 {
			continue // price still above trigger
		}
		logger.Infof("watcher: order %d TRIGGERED (price %s <= trigger %s) — settling", order.ID, price, order.TriggerPrice)
		if w.settleWithRetry(ctx, order) {
			w.store.MarkExecuted(order.ID)
			logger.Infof("watcher: order %d marked executed", order.ID)
		} else {
			logger.Warnf("watcher: order %d settle failed after %d attempts, will retry next tick", order.ID, settleAttempts)
		}
	}
	return nil
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

// settleWithRetry tries to settle one order up to settleAttempts times with
// exponential backoff. Returns true once a successful receipt is observed.
// This reveals the trigger price on-chain — by design: settlement is the
// moment of disclosure, and the vault re-verifies it against the live FTSO.
func (w *Watcher) settleWithRetry(ctx context.Context, order Order) bool {
	for attempt := 1; attempt <= settleAttempts; attempt++ {
		logger.Infof("watcher: order %d settle attempt %d/%d", order.ID, attempt, settleAttempts)
		err := w.settleOnce(ctx, order)
		if err == nil {
			return true
		}
		logger.Warnf("watcher: order %d settle attempt %d failed: %v", order.ID, attempt, err)
		if attempt < settleAttempts {
			backoff := time.Second << (attempt - 1) // 1s, 2s
			logger.Infof("watcher: order %d retrying in %s", order.ID, backoff)
			w.sleep(backoff)
		}
	}
	return false
}

// settleOnce builds, signs, sends one settle transaction and waits for its
// receipt.
func (w *Watcher) settleOnce(ctx context.Context, order Order) error {
	data, err := packSettleCalldata(order.ID, order.TriggerPrice, settleMaxAgeSec)
	if err != nil {
		return fmt.Errorf("packing settle calldata: %w", err)
	}

	if w.chainID == nil {
		chainID, err := w.client.ChainID(ctx)
		if err != nil {
			return fmt.Errorf("fetching chain id: %w", err)
		}
		w.chainID = chainID
	}

	nonce, err := w.client.PendingNonceAt(ctx, w.from)
	if err != nil {
		return fmt.Errorf("fetching nonce: %w", err)
	}
	gasPrice, err := w.client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("fetching gas price: %w", err)
	}
	gas, err := w.client.EstimateGas(ctx, ethereum.CallMsg{From: w.from, To: &w.vault, Data: data})
	if err != nil {
		return fmt.Errorf("estimating gas: %w", err)
	}

	tx := ethtypes.NewTransaction(nonce, w.vault, big.NewInt(0), gas, gasPrice, data)
	signed, err := ethtypes.SignTx(tx, ethtypes.LatestSignerForChainID(w.chainID), w.key)
	if err != nil {
		return fmt.Errorf("signing settle tx: %w", err)
	}

	if err := w.client.SendTransaction(ctx, signed); err != nil {
		return fmt.Errorf("sending settle tx: %w", err)
	}
	logger.Infof("watcher: order %d settle tx sent: %s (nonce %d, gas %d, gasPrice %s)",
		order.ID, signed.Hash(), nonce, gas, gasPrice)

	receipt, err := w.waitReceipt(ctx, signed.Hash())
	if err != nil {
		return fmt.Errorf("waiting for receipt: %w", err)
	}
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("settle tx %s reverted on-chain", signed.Hash())
	}
	logger.Infof("watcher: order %d settle confirmed in tx %s", order.ID, signed.Hash())
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

// waitReceipt polls for a transaction receipt until it lands or the poll
// budget runs out.
func (w *Watcher) waitReceipt(ctx context.Context, txHash common.Hash) (*ethtypes.Receipt, error) {
	for i := 0; i < receiptPolls; i++ {
		receipt, err := w.client.TransactionReceipt(ctx, txHash)
		if err == nil {
			return receipt, nil
		}
		if !errors.Is(err, ethereum.NotFound) {
			return nil, err
		}
		w.sleep(receiptPollDelay)
	}
	return nil, fmt.Errorf("tx %s: no receipt after %d polls", txHash, receiptPolls)
}
