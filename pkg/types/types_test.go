package types

import (
	"encoding/json"
	"math/big"
	"testing"

	"extension-scaffold/pkg/decoder"
)

// packPlaceOrder ABI-encodes (uint256 orderId, bytes ciphertext) exactly as the
// vault does: abi.encode(id, ciphertext) — two top-level arguments, NOT a tuple.
func packPlaceOrder(t *testing.T, orderID *big.Int, ciphertext []byte) []byte {
	t.Helper()
	data, err := PlaceOrderMessageArgs.Pack(orderID, ciphertext)
	if err != nil {
		t.Fatalf("packing place order: %v", err)
	}
	return data
}

// packCancelOrder ABI-encodes (uint256 orderId) as the vault's abi.encode(id).
func packCancelOrder(t *testing.T, orderID *big.Int) []byte {
	t.Helper()
	data, err := CancelOrderMessageArgs.Pack(orderID)
	if err != nil {
		t.Fatalf("packing cancel order: %v", err)
	}
	return data
}

func TestDecodePlaceOrder_RoundTrip(t *testing.T) {
	ciphertext := []byte{0xde, 0xad, 0xbe, 0xef, 0x01, 0x02, 0x03, 0x04, 0x05}
	data := packPlaceOrder(t, big.NewInt(7), ciphertext)

	req, err := DecodePlaceOrder(data)
	if err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if req.OrderID.Cmp(big.NewInt(7)) != 0 {
		t.Errorf("expected orderId 7, got %s", req.OrderID)
	}
	if string(req.Ciphertext) != string(ciphertext) {
		t.Errorf("ciphertext mismatch: got %x", req.Ciphertext)
	}
}

func TestDecodePlaceOrder_BadData(t *testing.T) {
	for name, data := range map[string][]byte{
		"empty":     {},
		"truncated": {0x00, 0x01, 0x02},
		"cancel encoding (missing bytes arg)": packCancelOrder(t, big.NewInt(7)),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodePlaceOrder(data); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestDecodeCancelOrder_RoundTrip(t *testing.T) {
	data := packCancelOrder(t, big.NewInt(42))

	req, err := DecodeCancelOrder(data)
	if err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if req.OrderID.Cmp(big.NewInt(42)) != 0 {
		t.Errorf("expected orderId 42, got %s", req.OrderID)
	}
}

func TestDecodeCancelOrder_BadData(t *testing.T) {
	for name, data := range map[string][]byte{
		"empty":     {},
		"truncated": {0x01, 0x02},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeCancelOrder(data); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestRegisterDecoders_PlaceOrderMessage(t *testing.T) {
	r := decoder.NewRegistry()
	RegisterDecoders(r)

	dec, err := r.Lookup("DARKSTOP", "PLACE_ORDER", decoder.KindMessage)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}

	ciphertext := []byte{0xaa, 0xbb, 0xcc}
	decoded, err := dec.Decode(packPlaceOrder(t, big.NewInt(3), ciphertext))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	req, ok := decoded.(PlaceOrderRequest)
	if !ok {
		t.Fatalf("expected PlaceOrderRequest, got %T", decoded)
	}
	if req.OrderID.Cmp(big.NewInt(3)) != 0 {
		t.Errorf("expected orderId 3, got %s", req.OrderID)
	}
	if string(req.Ciphertext) != string(ciphertext) {
		t.Errorf("ciphertext mismatch: got %x", req.Ciphertext)
	}
}

func TestRegisterDecoders_CancelOrderMessage(t *testing.T) {
	r := decoder.NewRegistry()
	RegisterDecoders(r)

	dec, err := r.Lookup("DARKSTOP", "CANCEL_ORDER", decoder.KindMessage)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}

	decoded, err := dec.Decode(packCancelOrder(t, big.NewInt(9)))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	req, ok := decoded.(CancelOrderRequest)
	if !ok {
		t.Fatalf("expected CancelOrderRequest, got %T", decoded)
	}
	if req.OrderID.Cmp(big.NewInt(9)) != 0 {
		t.Errorf("expected orderId 9, got %s", req.OrderID)
	}
}

func TestRegisterDecoders_Results(t *testing.T) {
	r := decoder.NewRegistry()
	RegisterDecoders(r)

	for _, cmd := range []string{"PLACE_ORDER", "CANCEL_ORDER"} {
		dec, err := r.Lookup("DARKSTOP", cmd, decoder.KindResult)
		if err != nil {
			t.Fatalf("lookup %s result: %v", cmd, err)
		}
		decoded, err := dec.Decode([]byte(`{"orderId":"1","status":"open"}`))
		if err != nil {
			t.Fatalf("decode %s result: %v", cmd, err)
		}
		resp, ok := decoded.(OrderResponse)
		if !ok {
			t.Fatalf("expected OrderResponse, got %T", decoded)
		}
		if resp.OrderID != "1" || resp.Status != "open" {
			t.Errorf("unexpected response: %+v", resp)
		}
	}
}

func TestRegisterDecoders_GreetingGone(t *testing.T) {
	r := decoder.NewRegistry()
	RegisterDecoders(r)

	if _, err := r.Lookup("GREETING", "SAY_HELLO", decoder.KindMessage); err == nil {
		t.Error("expected GREETING SAY_HELLO decoder to be unregistered")
	}
	if _, err := r.Lookup("GREETING", "SAY_GOODBYE", decoder.KindMessage); err == nil {
		t.Error("expected GREETING SAY_GOODBYE decoder to be unregistered")
	}
}

// State must expose the encryption pubkey and order list — and OrderState must
// contain ONLY orderId and status: no price data may ever appear in state.
func TestStateJSON_NoPriceData(t *testing.T) {
	s := State{
		EncryptionPubKey: "0x04abcd",
		OpenOrders:       1,
		Orders:           []OrderState{{OrderID: "1", Status: "open"}},
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"encryptionPubKey", "openOrders", "orders"} {
		if _, ok := m[key]; !ok {
			t.Errorf("expected state key %q in %s", key, b)
		}
	}

	ob, _ := json.Marshal(OrderState{OrderID: "1", Status: "open"})
	want := `{"orderId":"1","status":"open"}`
	if string(ob) != want {
		t.Errorf("OrderState JSON must be exactly %s (no extra fields), got %s", want, ob)
	}
}
