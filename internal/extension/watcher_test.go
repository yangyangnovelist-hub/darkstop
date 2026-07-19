package extension

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// testExecutorKey is a throwaway dev key used only in unit tests.
const testExecutorKey = "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"

var (
	testVaultAddr = common.HexToAddress("0x1000000000000000000000000000000000000001")
	testFtsoAddr  = common.HexToAddress("0x2000000000000000000000000000000000000002")
)

// mockClient is an in-memory ChainClient double.
type mockClient struct {
	mu sync.Mutex

	feedValue    *big.Int
	feedDecimals int8
	feedTS       uint64

	sendErrs      []error // consumed per SendTransaction call; nil = success
	sendCalls     int
	sent          []*ethtypes.Transaction
	receiptStatus uint64
	notFoundFirst int // TransactionReceipt returns NotFound this many times
	receiptCalls  int
	estimateErr   error
	estimateErrs  []error // consumed per EstimateGas call; nil = success
	estimateCalls int
	txLookupGone  bool
	txLookupMined bool
	orderStatus   uint8
	pendingNonce  uint64
	nonceCalls    int
}

func newMockClient() *mockClient {
	return &mockClient{
		feedValue:     big.NewInt(200_000), // 0.02 USD at 7 decimals
		feedDecimals:  7,
		feedTS:        uint64(time.Now().Unix()),
		receiptStatus: ethtypes.ReceiptStatusSuccessful,
		orderStatus:   1,
		pendingNonce:  7,
	}
}

func (m *mockClient) CallContract(_ context.Context, msg ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch *msg.To {
	case testVaultAddr:
		if len(msg.Data) < 4 {
			return nil, fmt.Errorf("short vault calldata")
		}
		switch {
		case bytes.Equal(msg.Data[:4], watcherABI.Methods["FTSO_V2"].ID):
			return watcherABI.Methods["FTSO_V2"].Outputs.Pack(testFtsoAddr)
		case bytes.Equal(msg.Data[:4], watcherABI.Methods["orders"].ID):
			return watcherABI.Methods["orders"].Outputs.Pack(common.HexToAddress("0x3000000000000000000000000000000000000003"), big.NewInt(1), m.orderStatus)
		default:
			return nil, fmt.Errorf("unexpected vault selector %x", msg.Data[:4])
		}
	case testFtsoAddr:
		return watcherABI.Methods["getFeedById"].Outputs.Pack(m.feedValue, m.feedDecimals, m.feedTS)
	default:
		return nil, fmt.Errorf("unexpected call target %s", msg.To)
	}
}

func (m *mockClient) SuggestGasPrice(context.Context) (*big.Int, error) {
	return big.NewInt(25_000_000_000), nil
}

func (m *mockClient) PendingNonceAt(context.Context, common.Address) (uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nonceCalls++
	return m.pendingNonce, nil
}

func (m *mockClient) EstimateGas(context.Context, ethereum.CallMsg) (uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.estimateCalls < len(m.estimateErrs) {
		err := m.estimateErrs[m.estimateCalls]
		m.estimateCalls++
		if err != nil {
			return 0, err
		}
		return 200_000, nil
	}
	m.estimateCalls++
	if m.estimateErr != nil {
		return 0, m.estimateErr
	}
	return 200_000, nil
}

func (m *mockClient) ChainID(context.Context) (*big.Int, error) {
	return big.NewInt(114), nil // Coston2
}

func (m *mockClient) SendTransaction(_ context.Context, tx *ethtypes.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var err error
	if m.sendCalls < len(m.sendErrs) {
		err = m.sendErrs[m.sendCalls]
	}
	m.sendCalls++
	if err != nil {
		return err
	}
	m.sent = append(m.sent, tx)
	return nil
}

func (m *mockClient) TransactionReceipt(_ context.Context, txHash common.Hash) (*ethtypes.Receipt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.receiptCalls++
	if m.receiptCalls <= m.notFoundFirst {
		return nil, ethereum.NotFound
	}
	return &ethtypes.Receipt{Status: m.receiptStatus, TxHash: txHash}, nil
}

func (m *mockClient) TransactionByHash(_ context.Context, txHash common.Hash) (*ethtypes.Transaction, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.txLookupGone {
		return nil, false, ethereum.NotFound
	}
	for _, tx := range m.sent {
		if tx.Hash() == txHash {
			return tx, !m.txLookupMined, nil
		}
	}
	return nil, false, ethereum.NotFound
}

// newTestWatcher builds a watcher over the mock with sleeping disabled.
func newTestWatcher(t *testing.T, client *mockClient, store *Store) *Watcher {
	t.Helper()
	w, err := NewWatcher(client, store, testVaultAddr, testExecutorKey, time.Second)
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	w.sleep = func(time.Duration) {}
	return w
}

func settleTriggerFromTx(t *testing.T, tx *ethtypes.Transaction) *big.Int {
	t.Helper()
	if len(tx.Data()) < 4 || !bytes.Equal(tx.Data()[:4], watcherABI.Methods["settle"].ID) {
		t.Fatalf("transaction is not settle calldata: %x", tx.Data())
	}
	values, err := watcherABI.Methods["settle"].Inputs.Unpack(tx.Data()[4:])
	if err != nil {
		t.Fatalf("decode settle calldata: %v", err)
	}
	trigger, ok := values[1].(*big.Int)
	if !ok {
		t.Fatalf("decoded trigger has type %T", values[1])
	}
	return trigger
}

func TestNewWatcher_BadKey(t *testing.T) {
	if _, err := NewWatcher(newMockClient(), NewStore(), testVaultAddr, "nonsense", time.Second); err == nil {
		t.Error("expected error for invalid executor key")
	}
	if _, err := NewWatcher(newMockClient(), NewStore(), testVaultAddr, "", time.Second); err == nil {
		t.Error("expected error for empty executor key")
	}
}

func TestFlrUsdFeedID(t *testing.T) {
	want := "01464c522f55534400000000000000000000000000"
	if got := hex.EncodeToString(flrUsdFeedID[:]); got != want {
		t.Errorf("feed id mismatch:\n got %s\nwant %s", got, want)
	}
}

func TestNormalizePrice(t *testing.T) {
	cases := []struct {
		value    int64
		decimals int8
		want     int64
	}{
		{200_000, 7, 20_000},   // 0.02 USD, feed at 7 decimals
		{2_000, 5, 20_000},     // 0.02 USD, feed at 5 decimals
		{20_000, 6, 20_000},    // already 6 decimals
		{2, 2, 20_000},         // 0.02 USD at 2 decimals
		{1_234_567, 8, 12_345}, // truncating division, 8 → 6 decimals
	}
	for _, tc := range cases {
		got := normalizePrice(big.NewInt(tc.value), tc.decimals)
		if got.Cmp(big.NewInt(tc.want)) != 0 {
			t.Errorf("normalizePrice(%d, %d) = %s, want %d", tc.value, tc.decimals, got, tc.want)
		}
	}
}

func TestWatcher_TickSettlesTriggeredOrder(t *testing.T) {
	client := newMockClient() // price 20_000 (6 dec)
	store := NewStore()
	_ = store.Put(Order{ID: 1, TriggerPrice: big.NewInt(20_000), Status: StatusOpen}) // price == trigger → settle
	w := newTestWatcher(t, client, store)

	if err := w.tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}

	if len(client.sent) != 1 {
		t.Fatalf("expected 1 settle tx, got %d", len(client.sent))
	}
	tx := client.sent[0]
	if *tx.To() != testVaultAddr {
		t.Errorf("settle tx sent to %s, want vault %s", tx.To(), testVaultAddr)
	}
	wantData, err := packSettleCalldata(1, big.NewInt(20_000), settleMaxAgeSec)
	if err != nil {
		t.Fatalf("packing expected calldata: %v", err)
	}
	if hex.EncodeToString(tx.Data()) != hex.EncodeToString(wantData) {
		t.Errorf("settle calldata mismatch:\n got %x\nwant %x", tx.Data(), wantData)
	}

	order, _ := store.Get(1)
	if order.Status != StatusExecuted {
		t.Errorf("expected order executed, got %s", order.Status)
	}
}

func TestWatcher_TickSkipsUntriggeredOrder(t *testing.T) {
	client := newMockClient() // price 20_000
	store := NewStore()
	_ = store.Put(Order{ID: 1, TriggerPrice: big.NewInt(19_999), Status: StatusOpen}) // trigger below price
	w := newTestWatcher(t, client, store)

	if err := w.tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}

	if client.sendCalls != 0 {
		t.Errorf("expected no settle attempts, got %d", client.sendCalls)
	}
	order, _ := store.Get(1)
	if order.Status != StatusOpen {
		t.Errorf("expected order still open, got %s", order.Status)
	}
}

func TestWatcher_TickSettlesOnlyTriggeredAmongMany(t *testing.T) {
	client := newMockClient() // price 20_000
	store := NewStore()
	_ = store.Put(Order{ID: 1, TriggerPrice: big.NewInt(25_000), Status: StatusOpen}) // triggered
	_ = store.Put(Order{ID: 2, TriggerPrice: big.NewInt(10_000), Status: StatusOpen}) // not triggered
	w := newTestWatcher(t, client, store)

	if err := w.tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}

	if len(client.sent) != 1 {
		t.Fatalf("expected exactly 1 settle tx, got %d", len(client.sent))
	}
	first, _ := store.Get(1)
	second, _ := store.Get(2)
	if first.Status != StatusExecuted {
		t.Errorf("expected order 1 executed, got %s", first.Status)
	}
	if second.Status != StatusOpen {
		t.Errorf("expected order 2 open, got %s", second.Status)
	}
}

func TestWatcher_TickSkipsStaleFeed(t *testing.T) {
	client := newMockClient()
	client.feedTS = uint64(time.Now().Add(-10 * time.Minute).Unix()) // older than settleMaxAgeSec
	store := NewStore()
	_ = store.Put(Order{ID: 1, TriggerPrice: big.NewInt(25_000), Status: StatusOpen})
	w := newTestWatcher(t, client, store)

	if err := w.tick(context.Background()); err == nil {
		t.Error("expected stale-feed error from tick")
	}
	if client.sendCalls != 0 {
		t.Errorf("expected no settle attempts against a stale feed, got %d", client.sendCalls)
	}
}

// A pre-broadcast RPC hiccup (here gas estimation) is safe to retry in-tick:
// nothing was signed, sent, or nonce-reserved yet. The watcher backs off
// exponentially and, once the transient error clears, settles with a single
// reserved nonce. Send/broadcast errors are deliberately NOT driven through
// this loop anymore — an uncertain broadcast is tracked as pending and
// reconciled across ticks (see TestWatcher_ReconcilesReceiptTimeoutWithoutDuplicateSend
// and TestWatcher_ReplacesDroppedTransactionAfterVaultReconciliation) so the
// executor never blindly re-broadcasts a transaction that may already be in the
// mempool.
func TestWatcher_RetriesTransientFailureWithBackoffThenSucceeds(t *testing.T) {
	client := newMockClient()
	client.estimateErrs = []error{errors.New("connection refused"), errors.New("connection refused"), nil}
	store := NewStore()
	_ = store.Put(Order{ID: 1, TriggerPrice: big.NewInt(25_000), Status: StatusOpen})

	var slept []time.Duration
	w := newTestWatcher(t, client, store)
	w.sleep = func(d time.Duration) { slept = append(slept, d) }

	if err := w.tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}

	if client.sendCalls != 1 || len(client.sent) != 1 {
		t.Fatalf("expected exactly one broadcast after transient retries, got sendCalls=%d sent=%d", client.sendCalls, len(client.sent))
	}
	if client.sent[0].Nonce() != 7 || client.nonceCalls != 1 {
		t.Fatalf("retries must reserve exactly one nonce; nonce=%d nonceCalls=%d", client.sent[0].Nonce(), client.nonceCalls)
	}
	order, _ := store.Get(1)
	if order.Status != StatusExecuted {
		t.Errorf("expected order executed after retry success, got %s", order.Status)
	}
	// Exponential backoff between failed attempts: strictly increasing waits.
	if len(slept) < 2 {
		t.Fatalf("expected at least 2 backoff sleeps, got %d", len(slept))
	}
	if slept[1] <= slept[0] {
		t.Errorf("expected growing backoff, got %v then %v", slept[0], slept[1])
	}
}

// An on-chain revert is a retryable, non-broadcast failure: the tx mined and
// consumed its nonce, so the watcher backs off and re-settles with a FRESH
// nonce, up to settleAttempts, then gives up for this tick leaving the order
// open for the next one. (A send/broadcast error no longer drives this loop —
// it is tracked as pending and reconciled instead of blindly re-sent.)
func TestWatcher_GivesUpAfterThreeFailures(t *testing.T) {
	client := newMockClient()
	client.receiptStatus = ethtypes.ReceiptStatusFailed // every settle reverts on-chain
	store := NewStore()
	_ = store.Put(Order{ID: 1, TriggerPrice: big.NewInt(25_000), Status: StatusOpen})

	var slept []time.Duration
	w := newTestWatcher(t, client, store)
	w.sleep = func(d time.Duration) { slept = append(slept, d) }

	_ = w.tick(context.Background())

	if client.sendCalls != settleAttempts {
		t.Errorf("expected exactly %d settle attempts, got %d", settleAttempts, client.sendCalls)
	}
	// Each reverted tx consumed its nonce, so distinct nonces are used per attempt.
	if len(client.sent) == settleAttempts && client.sent[0].Nonce() == client.sent[settleAttempts-1].Nonce() {
		t.Errorf("expected distinct nonces across reverted attempts, both were %d", client.sent[0].Nonce())
	}
	if len(slept) < 2 || slept[1] <= slept[0] {
		t.Errorf("expected growing backoff between attempts, got %v", slept)
	}
	// Order stays open: the next tick will try again.
	order, _ := store.Get(1)
	if order.Status != StatusOpen {
		t.Errorf("expected order still open after exhausted retries, got %s", order.Status)
	}
}

func TestWatcher_RevertedReceiptDoesNotExecute(t *testing.T) {
	client := newMockClient()
	client.receiptStatus = ethtypes.ReceiptStatusFailed
	store := NewStore()
	_ = store.Put(Order{ID: 1, TriggerPrice: big.NewInt(25_000), Status: StatusOpen})
	w := newTestWatcher(t, client, store)

	_ = w.tick(context.Background())

	order, _ := store.Get(1)
	if order.Status != StatusOpen {
		t.Errorf("expected order still open after reverted settle, got %s", order.Status)
	}
}

func TestWatcher_ReconcilesLateReceiptWithoutBlockingTicks(t *testing.T) {
	client := newMockClient()
	client.notFoundFirst = 3 // receipt appears on the 4th tick
	store := NewStore()
	_ = store.Put(Order{ID: 1, TriggerPrice: big.NewInt(25_000), Status: StatusOpen})
	w := newTestWatcher(t, client, store)

	for i := 0; i < 4; i++ {
		if err := w.tick(context.Background()); err != nil {
			t.Fatalf("tick %d: %v", i+1, err)
		}
	}

	order, _ := store.Get(1)
	if order.Status != StatusExecuted {
		t.Errorf("expected order executed once receipt lands, got %s", order.Status)
	}
	if client.receiptCalls != 4 {
		t.Errorf("expected one non-blocking receipt lookup per tick, got %d", client.receiptCalls)
	}
	if client.sendCalls != 1 {
		t.Errorf("expected one broadcast while receipt was pending, got %d", client.sendCalls)
	}
}

func TestWatcher_ReconcilesReceiptTimeoutWithoutDuplicateSend(t *testing.T) {
	client := newMockClient()
	client.notFoundFirst = 1
	store := NewStore()
	_ = store.Put(Order{ID: 1, TriggerPrice: big.NewInt(25_000), Status: StatusOpen})
	w := newTestWatcher(t, client, store)

	if err := w.tick(context.Background()); err != nil {
		t.Fatalf("first tick: %v", err)
	}
	if client.sendCalls != 1 {
		t.Fatalf("expected one broadcast before receipt timeout, got %d", client.sendCalls)
	}
	if _, ok := w.pending[1]; !ok {
		t.Fatal("expected timed-out receipt to remain pending by transaction hash")
	}
	order, _ := store.Get(1)
	if order.Status != StatusOpen {
		t.Fatalf("expected local order open while receipt is pending, got %s", order.Status)
	}

	if err := w.tick(context.Background()); err != nil {
		t.Fatalf("reconciliation tick: %v", err)
	}
	if client.sendCalls != 1 {
		t.Fatalf("expected no duplicate broadcast during reconciliation, got %d sends", client.sendCalls)
	}
	if _, ok := w.pending[1]; ok {
		t.Fatal("expected confirmed transaction to leave pending set")
	}
	order, _ = store.Get(1)
	if order.Status != StatusExecuted {
		t.Fatalf("expected reconciliation to mark order executed, got %s", order.Status)
	}
}

// Two orders trigger in the same tick and both confirm synchronously (receipts
// resolve in-tick, the mock default), so both settle within one tick.
// PendingNonceAt is pinned at 7 to model an RPC whose pending-nonce view lags
// behind the first broadcast; local reservation must still hand out 7 then 8
// rather than colliding on 7.
func TestWatcher_ReservesUniqueNoncesForMultipleBroadcastsInOneTick(t *testing.T) {
	client := newMockClient() // notFoundFirst=0: receipts resolve in-tick
	store := NewStore()
	_ = store.Put(Order{ID: 1, TriggerPrice: big.NewInt(25_000), Status: StatusOpen})
	_ = store.Put(Order{ID: 2, TriggerPrice: big.NewInt(25_000), Status: StatusOpen})
	w := newTestWatcher(t, client, store)

	if err := w.tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}
	if len(client.sent) != 2 {
		t.Fatalf("expected two broadcasts, got %d", len(client.sent))
	}
	if client.sent[0].Nonce() != 7 || client.sent[1].Nonce() != 8 {
		t.Fatalf("expected locally reserved nonces 7 and 8, got %d and %d", client.sent[0].Nonce(), client.sent[1].Nonce())
	}
	if client.nonceCalls != 2 {
		t.Fatalf("expected one RPC nonce sample per order, got %d", client.nonceCalls)
	}
	first, _ := store.Get(1)
	second, _ := store.Get(2)
	if first.Status != StatusExecuted || second.Status != StatusExecuted {
		t.Fatalf("expected both orders executed, got %s and %s", first.Status, second.Status)
	}
}

// With a single executor key the watcher keeps at most one UNCONFIRMED
// settlement in flight: a second triggered order waits until the first
// reconciles, then reuses the freed nonce floor. This prevents a stuck nonce-7
// tx from stranding a nonce-8 tx behind it, and proves serial settlements never
// collide on a nonce.
func TestWatcher_SerializesUnconfirmedSettlementsAcrossTicks(t *testing.T) {
	client := newMockClient()
	client.notFoundFirst = 1 // order 1's receipt is pending on tick 1, lands on tick 2
	store := NewStore()
	_ = store.Put(Order{ID: 1, TriggerPrice: big.NewInt(25_000), Status: StatusOpen})
	_ = store.Put(Order{ID: 2, TriggerPrice: big.NewInt(25_000), Status: StatusOpen})
	w := newTestWatcher(t, client, store)

	// Tick 1: order 1 broadcasts and is left pending; order 2 must not broadcast.
	if err := w.tick(context.Background()); err != nil {
		t.Fatalf("tick 1: %v", err)
	}
	if len(client.sent) != 1 {
		t.Fatalf("expected only order 1 to broadcast while it is unconfirmed, got %d sends", len(client.sent))
	}
	if _, ok := w.pending[1]; !ok {
		t.Fatal("order 1 should be tracked as pending after tick 1")
	}
	if s2, _ := store.Get(2); s2.Status != StatusOpen {
		t.Fatalf("order 2 must stay open while order 1 is unconfirmed, got %s", s2.Status)
	}

	// Tick 2: order 1's receipt lands and reconciles; order 2 now broadcasts.
	if err := w.tick(context.Background()); err != nil {
		t.Fatalf("tick 2: %v", err)
	}
	if len(client.sent) != 2 {
		t.Fatalf("expected order 2 to broadcast after order 1 confirmed, got %d sends", len(client.sent))
	}
	if client.sent[0].Nonce() == client.sent[1].Nonce() {
		t.Fatalf("serial settlements must not collide on nonce %d", client.sent[0].Nonce())
	}
	if s1, _ := store.Get(1); s1.Status != StatusExecuted {
		t.Fatalf("order 1 should be executed after its receipt landed, got %s", s1.Status)
	}
}

func TestWatcher_MinedWithoutReceiptUsesFreshNonce(t *testing.T) {
	client := newMockClient()
	client.notFoundFirst = 10_000
	store := NewStore()
	_ = store.Put(Order{ID: 1, TriggerPrice: big.NewInt(25_000), Status: StatusOpen})
	w := newTestWatcher(t, client, store)

	if err := w.tick(context.Background()); err != nil {
		t.Fatalf("initial tick: %v", err)
	}
	client.txLookupMined = true
	client.pendingNonce = 8
	if err := w.tick(context.Background()); err != nil {
		t.Fatalf("mined reconciliation tick: %v", err)
	}

	if len(client.sent) != 2 {
		t.Fatalf("expected a fresh settlement after mined revert, got %d broadcasts", len(client.sent))
	}
	if client.sent[0].Nonce() != 7 || client.sent[1].Nonce() != 8 {
		t.Fatalf("consumed nonce was reused: got %d then %d", client.sent[0].Nonce(), client.sent[1].Nonce())
	}
}

func TestWatcher_TrailingReplacementUsesLatestEffectiveTrigger(t *testing.T) {
	client := newMockClient()
	client.notFoundFirst = 10_000
	client.feedValue = big.NewInt(180_000) // 18_000 at 6dp
	store := NewStore()
	store.ObservePrice(big.NewInt(20_000), time.Now())
	if err := store.Put(Order{ID: 1, TrailBps: 500, Status: StatusOpen}); err != nil {
		t.Fatalf("put trailing order: %v", err)
	}
	w := newTestWatcher(t, client, store)

	if err := w.tick(context.Background()); err != nil {
		t.Fatalf("initial trigger tick: %v", err)
	}
	if got := settleTriggerFromTx(t, client.sent[0]); got.Cmp(big.NewInt(19_000)) != 0 {
		t.Fatalf("initial trailing trigger = %s, want 19000", got)
	}
	pending := w.pending[1]
	pending.firstSeen = time.Now().Add(-pendingReplacementAge - time.Second)
	w.pending[1] = pending

	client.feedValue = big.NewInt(240_000) // new high 24_000; trigger 22_800
	client.feedTS = uint64(time.Now().Unix())
	if err := w.tick(context.Background()); err != nil {
		t.Fatalf("new-high tick: %v", err)
	}
	if len(client.sent) != 1 {
		t.Fatalf("inactive trailing policy should not be fee-bumped, got %d sends", len(client.sent))
	}
	if _, ok := w.pending[1]; !ok {
		t.Fatal("known mempool transaction should remain tracked while policy is inactive")
	}

	client.feedValue = big.NewInt(228_000) // crosses the new 22_800 trigger
	client.feedTS = uint64(time.Now().Unix())
	if err := w.tick(context.Background()); err != nil {
		t.Fatalf("replacement tick: %v", err)
	}
	if len(client.sent) != 2 {
		t.Fatalf("expected one replacement, got %d sends", len(client.sent))
	}
	if client.sent[1].Nonce() != client.sent[0].Nonce() {
		t.Fatalf("replacement changed nonce: %d -> %d", client.sent[0].Nonce(), client.sent[1].Nonce())
	}
	if got := settleTriggerFromTx(t, client.sent[1]); got.Cmp(big.NewInt(22_800)) != 0 {
		t.Fatalf("replacement used stale trigger %s, want 22800", got)
	}
}

func TestWatcher_DroppedInactiveTrailingWaitsForFreshCrossing(t *testing.T) {
	client := newMockClient()
	client.notFoundFirst = 10_000
	client.feedValue = big.NewInt(180_000)
	store := NewStore()
	store.ObservePrice(big.NewInt(20_000), time.Now())
	if err := store.Put(Order{ID: 1, TrailBps: 500, Status: StatusOpen}); err != nil {
		t.Fatalf("put trailing order: %v", err)
	}
	w := newTestWatcher(t, client, store)

	if err := w.tick(context.Background()); err != nil {
		t.Fatalf("initial trigger tick: %v", err)
	}
	client.txLookupGone = true
	client.feedValue = big.NewInt(240_000)
	for i := 0; i < pendingDropChecks; i++ {
		client.feedTS = uint64(time.Now().Unix())
		if err := w.tick(context.Background()); err != nil {
			t.Fatalf("drop reconciliation tick %d: %v", i+1, err)
		}
	}
	if len(client.sent) != 1 {
		t.Fatalf("inactive dropped transaction should not be replaced, got %d sends", len(client.sent))
	}
	if _, ok := w.pending[1]; ok {
		t.Fatal("dropped inactive transaction should release pending ownership")
	}

	client.feedValue = big.NewInt(228_000)
	client.feedTS = uint64(time.Now().Unix())
	if err := w.tick(context.Background()); err != nil {
		t.Fatalf("fresh crossing tick: %v", err)
	}
	if len(client.sent) != 2 {
		t.Fatalf("expected fresh settlement on new crossing, got %d sends", len(client.sent))
	}
	if got := settleTriggerFromTx(t, client.sent[1]); got.Cmp(big.NewInt(22_800)) != 0 {
		t.Fatalf("fresh settlement trigger = %s, want 22800", got)
	}
}

func TestWatcher_ReplacesDroppedTransactionAfterVaultReconciliation(t *testing.T) {
	client := newMockClient()
	client.notFoundFirst = 10_000
	client.txLookupGone = true
	client.orderStatus = 1
	store := NewStore()
	_ = store.Put(Order{ID: 1, TriggerPrice: big.NewInt(25_000), Status: StatusOpen})
	w := newTestWatcher(t, client, store)

	if err := w.tick(context.Background()); err != nil {
		t.Fatalf("initial tick: %v", err)
	}
	if client.sendCalls != 1 {
		t.Fatalf("expected initial broadcast, got %d", client.sendCalls)
	}

	for i := 0; i < pendingDropChecks; i++ {
		if err := w.tick(context.Background()); err != nil {
			t.Fatalf("reconciliation tick %d: %v", i+1, err)
		}
	}
	if client.sendCalls != 2 {
		t.Fatalf("expected exactly one safe replacement after confirmed drop, got %d sends", client.sendCalls)
	}
	if client.sent[0].Nonce() != client.sent[1].Nonce() {
		t.Fatalf("replacement nonce changed: %d -> %d", client.sent[0].Nonce(), client.sent[1].Nonce())
	}
	if client.sent[1].GasPrice().Cmp(client.sent[0].GasPrice()) <= 0 {
		t.Fatalf("replacement gas price did not increase: %s -> %s", client.sent[0].GasPrice(), client.sent[1].GasPrice())
	}
	if _, ok := w.pending[1]; !ok {
		t.Fatal("expected replacement transaction to be tracked")
	}
}

func TestWatcher_FeeBumpsStuckMempoolTransaction(t *testing.T) {
	client := newMockClient()
	client.notFoundFirst = 10_000
	store := NewStore()
	_ = store.Put(Order{ID: 1, TriggerPrice: big.NewInt(25_000), Status: StatusOpen})
	w := newTestWatcher(t, client, store)

	_ = w.tick(context.Background())
	pending := w.pending[1]
	pending.firstSeen = time.Now().Add(-pendingReplacementAge - time.Second)
	w.pending[1] = pending

	if err := w.tick(context.Background()); err != nil {
		t.Fatalf("replacement tick: %v", err)
	}
	if client.sendCalls != 2 {
		t.Fatalf("expected one fee-bumped replacement, got %d sends", client.sendCalls)
	}
	if client.sent[0].Nonce() != client.sent[1].Nonce() {
		t.Fatalf("fee bump must reuse nonce: %d -> %d", client.sent[0].Nonce(), client.sent[1].Nonce())
	}
	if client.sent[1].GasPrice().Cmp(client.sent[0].GasPrice()) <= 0 {
		t.Fatalf("fee bump did not increase gas price: %s -> %s", client.sent[0].GasPrice(), client.sent[1].GasPrice())
	}
}

func TestWatcher_RepairsExecutedChainStateWhenReceiptIsMissing(t *testing.T) {
	client := newMockClient()
	client.notFoundFirst = 10_000
	client.txLookupGone = true
	store := NewStore()
	_ = store.Put(Order{ID: 1, TriggerPrice: big.NewInt(25_000), Status: StatusOpen})
	w := newTestWatcher(t, client, store)

	_ = w.tick(context.Background())
	client.orderStatus = 2
	for i := 0; i < pendingDropChecks; i++ {
		w.reconcilePending(context.Background(), nil)
	}
	order, _ := store.Get(1)
	if order.Status != StatusExecuted {
		t.Fatalf("expected canonical chain state to repair local order, got %s", order.Status)
	}
	if client.sendCalls != 1 {
		t.Fatalf("expected no replacement after chain reports execution, got %d sends", client.sendCalls)
	}
}

func TestWatcher_EstimateGasFailureRetriesAndStaysOpen(t *testing.T) {
	client := newMockClient()
	client.estimateErr = errors.New("execution reverted: price above trigger")
	store := NewStore()
	_ = store.Put(Order{ID: 1, TriggerPrice: big.NewInt(25_000), Status: StatusOpen})
	w := newTestWatcher(t, client, store)

	_ = w.tick(context.Background())

	if client.sendCalls != 0 {
		t.Errorf("expected no raw sends when estimation fails, got %d", client.sendCalls)
	}
	order, _ := store.Get(1)
	if order.Status != StatusOpen {
		t.Errorf("expected order still open, got %s", order.Status)
	}
}

func TestWatcher_RunStopsOnContextCancel(t *testing.T) {
	client := newMockClient()
	w, err := NewWatcher(client, NewStore(), testVaultAddr, testExecutorKey, time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	w.sleep = func(time.Duration) {}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not stop on context cancel")
	}
}
