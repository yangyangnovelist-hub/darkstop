package extension

import (
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
}

func newMockClient() *mockClient {
	return &mockClient{
		feedValue:     big.NewInt(200_000), // 0.02 USD at 7 decimals
		feedDecimals:  7,
		feedTS:        uint64(time.Now().Unix()),
		receiptStatus: ethtypes.ReceiptStatusSuccessful,
	}
}

func (m *mockClient) CallContract(_ context.Context, msg ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch *msg.To {
	case testVaultAddr:
		return watcherABI.Methods["FTSO_V2"].Outputs.Pack(testFtsoAddr)
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
	return 7, nil
}

func (m *mockClient) EstimateGas(context.Context, ethereum.CallMsg) (uint64, error) {
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
		{200_000, 7, 20_000},  // 0.02 USD, feed at 7 decimals
		{2_000, 5, 20_000},    // 0.02 USD, feed at 5 decimals
		{20_000, 6, 20_000},   // already 6 decimals
		{2, 2, 20_000},        // 0.02 USD at 2 decimals
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

func TestWatcher_RetriesSendWithBackoffThenSucceeds(t *testing.T) {
	client := newMockClient()
	client.sendErrs = []error{errors.New("nonce too low"), errors.New("connection refused"), nil}
	store := NewStore()
	_ = store.Put(Order{ID: 1, TriggerPrice: big.NewInt(25_000), Status: StatusOpen})

	var slept []time.Duration
	w := newTestWatcher(t, client, store)
	w.sleep = func(d time.Duration) { slept = append(slept, d) }

	if err := w.tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}

	if client.sendCalls != 3 {
		t.Errorf("expected 3 send attempts, got %d", client.sendCalls)
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

func TestWatcher_GivesUpAfterThreeFailures(t *testing.T) {
	client := newMockClient()
	client.sendErrs = []error{errors.New("boom"), errors.New("boom"), errors.New("boom")}
	store := NewStore()
	_ = store.Put(Order{ID: 1, TriggerPrice: big.NewInt(25_000), Status: StatusOpen})
	w := newTestWatcher(t, client, store)

	_ = w.tick(context.Background())

	if client.sendCalls != 3 {
		t.Errorf("expected exactly 3 send attempts, got %d", client.sendCalls)
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

func TestWatcher_WaitsForLateReceipt(t *testing.T) {
	client := newMockClient()
	client.notFoundFirst = 3 // receipt appears on the 4th poll
	store := NewStore()
	_ = store.Put(Order{ID: 1, TriggerPrice: big.NewInt(25_000), Status: StatusOpen})
	w := newTestWatcher(t, client, store)

	if err := w.tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}

	order, _ := store.Get(1)
	if order.Status != StatusExecuted {
		t.Errorf("expected order executed once receipt lands, got %s", order.Status)
	}
	if client.receiptCalls < 4 {
		t.Errorf("expected at least 4 receipt polls, got %d", client.receiptCalls)
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
