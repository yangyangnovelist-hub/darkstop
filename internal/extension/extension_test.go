package extension

import (
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"extension-scaffold/internal/config"
	"extension-scaffold/pkg/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	teetypes "github.com/flare-foundation/tee-node/pkg/types"
	teeutils "github.com/flare-foundation/tee-node/pkg/utils"
)

// toHash mirrors teeutils.ToHash for clarity: left-pads a string into a 32-byte hash.
func toHash(s string) common.Hash { return teeutils.ToHash(s) }

// buildTestAction constructs a teetypes.Action whose Data.Message is the
// JSON-encoded DataFixed payload. This is what processAction expects to parse.
func buildTestAction(opType, opCommand common.Hash, originalMessage []byte) teetypes.Action {
	// DataFixed is the structure that processorutils.Parse extracts from Data.Message.
	type dataFixed struct {
		InstructionID      common.Hash    `json:"instructionId"`
		TeeID              common.Address `json:"teeId"`
		Timestamp          uint64         `json:"timestamp"`
		RewardEpochID      uint32         `json:"rewardEpochId"`
		OPType             common.Hash    `json:"opType"`
		OPCommand          common.Hash    `json:"opCommand"`
		Cosigners          []string       `json:"cosigners"`
		CosignersThreshold uint64         `json:"cosignersThreshold"`
		OriginalMessage    hexutil.Bytes  `json:"originalMessage"`
	}

	df := dataFixed{
		OPType:          opType,
		OPCommand:       opCommand,
		OriginalMessage: originalMessage,
	}
	msg, _ := json.Marshal(df)

	return teetypes.Action{
		Data: teetypes.ActionData{
			ID:            common.HexToHash("0x1234"),
			SubmissionTag: "submit",
			Message:       msg,
		},
	}
}

// newTestExtension builds an Extension with a fresh enclave keypair and store.
// The server is constructed but never started.
func newTestExtension(t *testing.T) *Extension {
	t.Helper()
	return New(0, 0)
}

// encryptTrigger encrypts a {"triggerPrice": ...} payload to the extension's
// own enclave key, as a browser client would with the pubkey from /state.
func encryptTrigger(t *testing.T, e *Extension, triggerPrice string) []byte {
	t.Helper()
	ct, err := e.crypto.Encrypt([]byte(`{"triggerPrice":"` + triggerPrice + `"}`))
	if err != nil {
		t.Fatalf("encrypting trigger: %v", err)
	}
	return ct
}

// packPlace ABI-encodes a PLACE_ORDER message: abi.encode(orderId, ciphertext).
func packPlace(t *testing.T, orderID *big.Int, ciphertext []byte) []byte {
	t.Helper()
	data, err := types.PlaceOrderMessageArgs.Pack(orderID, ciphertext)
	if err != nil {
		t.Fatalf("packing place message: %v", err)
	}
	return data
}

// packCancel ABI-encodes a CANCEL_ORDER message: abi.encode(orderId).
func packCancel(t *testing.T, orderID *big.Int) []byte {
	t.Helper()
	data, err := types.CancelOrderMessageArgs.Pack(orderID)
	if err != nil {
		t.Fatalf("packing cancel message: %v", err)
	}
	return data
}

// runAction pushes an action through processAction and returns the ActionResult.
func runAction(t *testing.T, e *Extension, action teetypes.Action) teetypes.ActionResult {
	t.Helper()
	status, body := e.processAction(action)
	if status != http.StatusOK {
		t.Fatalf("expected HTTP %d, got %d: %s", http.StatusOK, status, body)
	}
	var result teetypes.ActionResult
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshalling ActionResult: %v", err)
	}
	return result
}

// --- Routing ---

func TestProcessAction_UnknownOPType(t *testing.T) {
	e := newTestExtension(t)
	action := buildTestAction(toHash("UNKNOWN_TYPE"), toHash(config.OPCommandPlaceOrder), nil)

	status, body := e.processAction(action)

	if status != http.StatusNotImplemented {
		t.Fatalf("expected status %d, got %d", http.StatusNotImplemented, status)
	}
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "unsupported op type") {
		t.Error("expected body to contain 'unsupported op type'")
	}
	if !strings.Contains(bodyStr, toHash("UNKNOWN_TYPE").Hex()) {
		t.Error("expected body to contain the received hash")
	}
	if !strings.Contains(bodyStr, toHash(config.OPTypeDarkstop).Hex()) {
		t.Error("expected body to contain the expected DARKSTOP hash")
	}
	if !strings.Contains(bodyStr, config.OPTypeDarkstop) {
		t.Errorf("expected body to contain %q", config.OPTypeDarkstop)
	}
}

func TestProcessAction_UnknownOPCommand(t *testing.T) {
	e := newTestExtension(t)
	action := buildTestAction(toHash(config.OPTypeDarkstop), toHash("UNKNOWN_COMMAND"), nil)

	status, body := e.processAction(action)

	if status != http.StatusNotImplemented {
		t.Fatalf("expected status %d, got %d", http.StatusNotImplemented, status)
	}
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "unsupported op command") {
		t.Error("expected body to contain 'unsupported op command'")
	}
	for _, cmd := range []string{config.OPCommandPlaceOrder, config.OPCommandCancelOrder} {
		if !strings.Contains(bodyStr, toHash(cmd).Hex()) {
			t.Errorf("expected body to contain hash for %s", cmd)
		}
		if !strings.Contains(bodyStr, cmd) {
			t.Errorf("expected body to contain command name %q", cmd)
		}
	}
}

func TestProcessAction_InvalidDataMessage(t *testing.T) {
	e := newTestExtension(t)
	action := teetypes.Action{
		Data: teetypes.ActionData{
			ID:      common.HexToHash("0xabcd"),
			Message: []byte(`not json at all`),
		},
	}

	status, body := e.processAction(action)

	if status != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, status, body)
	}
	if !strings.Contains(string(body), "decoding fixed data") {
		t.Errorf("expected body to mention 'decoding fixed data', got %q", body)
	}
}

// --- PLACE_ORDER ---

func TestProcessPlaceOrder(t *testing.T) {
	cases := []struct {
		name       string
		message    func(t *testing.T, e *Extension) []byte
		before     func(t *testing.T, e *Extension) // optional store setup
		wantStatus uint8
		wantLog    string // substring of ActionResult.Log for errors
	}{
		{
			name: "happy path",
			message: func(t *testing.T, e *Extension) []byte {
				return packPlace(t, big.NewInt(1), encryptTrigger(t, e, "20000"))
			},
			wantStatus: 1,
		},
		{
			name: "bad ABI message",
			message: func(t *testing.T, e *Extension) []byte {
				return []byte{0x01, 0x02, 0x03}
			},
			wantStatus: 0,
			wantLog:    "decoding request",
		},
		{
			name: "bad ciphertext",
			message: func(t *testing.T, e *Extension) []byte {
				return packPlace(t, big.NewInt(1), []byte("this is not an ecies ciphertext"))
			},
			wantStatus: 0,
			wantLog:    "decrypt",
		},
		{
			name: "ciphertext for another key",
			message: func(t *testing.T, e *Extension) []byte {
				other, err := NewCrypto()
				if err != nil {
					t.Fatal(err)
				}
				ct, err := other.Encrypt([]byte(`{"triggerPrice":"20000"}`))
				if err != nil {
					t.Fatal(err)
				}
				return packPlace(t, big.NewInt(1), ct)
			},
			wantStatus: 0,
			wantLog:    "decrypt",
		},
		{
			name: "zero trigger",
			message: func(t *testing.T, e *Extension) []byte {
				return packPlace(t, big.NewInt(1), encryptTrigger(t, e, "0"))
			},
			wantStatus: 0,
			wantLog:    "positive",
		},
		{
			name: "malformed plaintext",
			message: func(t *testing.T, e *Extension) []byte {
				ct, err := e.crypto.Encrypt([]byte(`not json`))
				if err != nil {
					t.Fatal(err)
				}
				return packPlace(t, big.NewInt(1), ct)
			},
			wantStatus: 0,
			wantLog:    "trigger",
		},
		{
			name: "zero order id",
			message: func(t *testing.T, e *Extension) []byte {
				return packPlace(t, big.NewInt(0), encryptTrigger(t, e, "20000"))
			},
			wantStatus: 0,
			wantLog:    "order id",
		},
		{
			name: "order id exceeds uint64",
			message: func(t *testing.T, e *Extension) []byte {
				huge, _ := new(big.Int).SetString("18446744073709551616", 10) // 2^64
				return packPlace(t, huge, encryptTrigger(t, e, "20000"))
			},
			wantStatus: 0,
			wantLog:    "order id",
		},
		{
			name: "duplicate id",
			before: func(t *testing.T, e *Extension) {
				if err := e.store.Put(Order{ID: 1, TriggerPrice: big.NewInt(111), Status: StatusOpen}); err != nil {
					t.Fatal(err)
				}
			},
			message: func(t *testing.T, e *Extension) []byte {
				return packPlace(t, big.NewInt(1), encryptTrigger(t, e, "20000"))
			},
			wantStatus: 0,
			wantLog:    "already exists",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newTestExtension(t)
			if tc.before != nil {
				tc.before(t, e)
			}
			action := buildTestAction(
				toHash(config.OPTypeDarkstop),
				toHash(config.OPCommandPlaceOrder),
				tc.message(t, e),
			)

			result := runAction(t, e, action)

			if result.Status != tc.wantStatus {
				t.Fatalf("expected ActionResult.Status=%d, got %d (log: %s)", tc.wantStatus, result.Status, result.Log)
			}
			if tc.wantStatus == 0 {
				if !strings.Contains(result.Log, tc.wantLog) {
					t.Errorf("expected log to contain %q, got %q", tc.wantLog, result.Log)
				}
				return
			}

			var resp types.OrderResponse
			if err := json.Unmarshal(result.Data, &resp); err != nil {
				t.Fatalf("unmarshalling OrderResponse: %v", err)
			}
			if resp.OrderID != "1" || resp.Status != StatusOpen {
				t.Errorf("unexpected response: %+v", resp)
			}
			order, ok := e.store.Get(1)
			if !ok {
				t.Fatal("expected order 1 in store")
			}
			if order.TriggerPrice.Cmp(big.NewInt(20000)) != 0 {
				t.Errorf("expected stored trigger 20000, got %s", order.TriggerPrice)
			}
			if order.Status != StatusOpen {
				t.Errorf("expected stored status open, got %s", order.Status)
			}
		})
	}
}

func TestProcessPlaceOrder_ResponseNeverContainsTrigger(t *testing.T) {
	e := newTestExtension(t)
	action := buildTestAction(
		toHash(config.OPTypeDarkstop),
		toHash(config.OPCommandPlaceOrder),
		packPlace(t, big.NewInt(7), encryptTrigger(t, e, "31337")),
	)

	status, body := e.processAction(action)
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if strings.Contains(string(body), "31337") {
		t.Errorf("action result leaks trigger price: %s", body)
	}
}

// --- CANCEL_ORDER ---

func TestProcessCancelOrder(t *testing.T) {
	cases := []struct {
		name       string
		before     func(t *testing.T, e *Extension)
		message    func(t *testing.T) []byte
		wantStatus uint8
		wantLog    string
	}{
		{
			name: "happy path",
			before: func(t *testing.T, e *Extension) {
				if err := e.store.Put(Order{ID: 5, TriggerPrice: big.NewInt(20000), Status: StatusOpen}); err != nil {
					t.Fatal(err)
				}
			},
			message:    func(t *testing.T) []byte { return packCancel(t, big.NewInt(5)) },
			wantStatus: 1,
		},
		{
			name:       "unknown order",
			message:    func(t *testing.T) []byte { return packCancel(t, big.NewInt(99)) },
			wantStatus: 0,
			wantLog:    "not found",
		},
		{
			name:       "bad ABI message",
			message:    func(t *testing.T) []byte { return []byte{0xff} },
			wantStatus: 0,
			wantLog:    "decoding request",
		},
		{
			name:       "zero order id",
			message:    func(t *testing.T) []byte { return packCancel(t, big.NewInt(0)) },
			wantStatus: 0,
			wantLog:    "order id",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newTestExtension(t)
			if tc.before != nil {
				tc.before(t, e)
			}
			action := buildTestAction(
				toHash(config.OPTypeDarkstop),
				toHash(config.OPCommandCancelOrder),
				tc.message(t),
			)

			result := runAction(t, e, action)

			if result.Status != tc.wantStatus {
				t.Fatalf("expected ActionResult.Status=%d, got %d (log: %s)", tc.wantStatus, result.Status, result.Log)
			}
			if tc.wantStatus == 0 {
				if !strings.Contains(result.Log, tc.wantLog) {
					t.Errorf("expected log to contain %q, got %q", tc.wantLog, result.Log)
				}
				return
			}

			var resp types.OrderResponse
			if err := json.Unmarshal(result.Data, &resp); err != nil {
				t.Fatalf("unmarshalling OrderResponse: %v", err)
			}
			if resp.OrderID != "5" || resp.Status != StatusCancelled {
				t.Errorf("unexpected response: %+v", resp)
			}
			if _, ok := e.store.Get(5); ok {
				t.Error("expected cancelled order to be dropped from the store")
			}
		})
	}
}

// --- POST /action over real HTTP ---

func TestActionHandler_PlaceOrderOverHTTP(t *testing.T) {
	e := newTestExtension(t)
	server := httptest.NewServer(e.Server.Handler)
	defer server.Close()

	action := buildTestAction(
		toHash(config.OPTypeDarkstop),
		toHash(config.OPCommandPlaceOrder),
		packPlace(t, big.NewInt(1), encryptTrigger(t, e, "20000")),
	)
	body, _ := json.Marshal(action)

	httpResp, err := http.Post(server.URL+"/action", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("POST /action: %v", err)
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", httpResp.StatusCode)
	}

	var result teetypes.ActionResult
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		t.Fatalf("decoding ActionResult: %v", err)
	}
	if result.Status != 1 {
		t.Fatalf("expected success, got status %d (log: %s)", result.Status, result.Log)
	}
	var resp types.OrderResponse
	if err := json.Unmarshal(result.Data, &resp); err != nil {
		t.Fatalf("decoding OrderResponse: %v", err)
	}
	if resp.OrderID != "1" || resp.Status != StatusOpen {
		t.Errorf("unexpected response: %+v", resp)
	}
}

// --- GET /state ---

func TestStateHandler(t *testing.T) {
	e := newTestExtension(t)
	if err := e.store.Put(Order{ID: 1, TriggerPrice: big.NewInt(31337), Status: StatusOpen}); err != nil {
		t.Fatal(err)
	}
	if err := e.store.Put(Order{ID: 2, TriggerPrice: big.NewInt(42424242), Status: StatusOpen}); err != nil {
		t.Fatal(err)
	}
	e.store.MarkExecuted(2)

	rec := httptest.NewRecorder()
	e.stateHandler(rec, httptest.NewRequest("GET", "/state", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	// Trigger prices must NEVER appear in state.
	for _, secret := range []string{"31337", "42424242"} {
		if strings.Contains(body, secret) {
			t.Errorf("state leaks trigger price %s: %s", secret, body)
		}
	}

	var resp types.StateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshalling state: %v", err)
	}
	if resp.State.EncryptionPubKey != e.crypto.PublicKeyHex() {
		t.Errorf("expected pubkey %s, got %s", e.crypto.PublicKeyHex(), resp.State.EncryptionPubKey)
	}
	if resp.State.OpenOrders != 1 {
		t.Errorf("expected 1 open order, got %d", resp.State.OpenOrders)
	}
	if len(resp.State.Orders) != 2 {
		t.Fatalf("expected 2 orders in state, got %d", len(resp.State.Orders))
	}
	if resp.State.Orders[0].OrderID != "1" || resp.State.Orders[0].Status != StatusOpen {
		t.Errorf("unexpected first order: %+v", resp.State.Orders[0])
	}
	if resp.State.Orders[1].OrderID != "2" || resp.State.Orders[1].Status != StatusExecuted {
		t.Errorf("unexpected second order: %+v", resp.State.Orders[1])
	}
	if resp.StateVersion != teeutils.ToHash(config.Version) {
		t.Errorf("unexpected state version %s", resp.StateVersion.Hex())
	}
}
