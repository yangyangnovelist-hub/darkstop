// Package types contains types that could be useful to other apps when interacting with this extension.
package types

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// PlaceOrderRequest is the ABI-decoded payload of a PLACE_ORDER instruction.
// It mirrors the vault's `abi.encode(uint256 orderId, bytes ciphertext)` —
// two top-level arguments, NOT a single tuple. The ciphertext is the user's
// ECIES-encrypted trigger parameters, opaque to everyone but the TEE.
type PlaceOrderRequest struct {
	OrderID    *big.Int `json:"orderId"`
	Ciphertext []byte   `json:"ciphertext"`
}

// CancelOrderRequest is the ABI-decoded payload of a CANCEL_ORDER instruction.
// It mirrors the vault's `abi.encode(uint256 orderId)`.
type CancelOrderRequest struct {
	OrderID *big.Int `json:"orderId"`
}

// OrderResponse is the JSON payload returned in ActionResult.Data for both
// PLACE_ORDER and CANCEL_ORDER. It carries NO price data.
type OrderResponse struct {
	OrderID string `json:"orderId"`
	Status  string `json:"status"`
}

// ABI argument layouts matching the vault's instruction messages.
var (
	// PlaceOrderMessageArgs is abi.encode(uint256 orderId, bytes ciphertext).
	PlaceOrderMessageArgs abi.Arguments
	// CancelOrderMessageArgs is abi.encode(uint256 orderId).
	CancelOrderMessageArgs abi.Arguments
)

func init() {
	uint256Ty, _ := abi.NewType("uint256", "", nil)
	bytesTy, _ := abi.NewType("bytes", "", nil)
	PlaceOrderMessageArgs = abi.Arguments{
		{Name: "orderId", Type: uint256Ty},
		{Name: "ciphertext", Type: bytesTy},
	}
	CancelOrderMessageArgs = abi.Arguments{
		{Name: "orderId", Type: uint256Ty},
	}
}

// DecodePlaceOrder decodes abi.encode(uint256 orderId, bytes ciphertext).
func DecodePlaceOrder(data []byte) (PlaceOrderRequest, error) {
	var req PlaceOrderRequest
	values, err := unpackStrict(PlaceOrderMessageArgs, data)
	if err != nil {
		return req, fmt.Errorf("unpacking place order message: %w", err)
	}
	orderID, ok := values[0].(*big.Int)
	if !ok {
		return req, fmt.Errorf("orderId: expected *big.Int, got %T", values[0])
	}
	ciphertext, ok := values[1].([]byte)
	if !ok {
		return req, fmt.Errorf("ciphertext: expected []byte, got %T", values[1])
	}
	req.OrderID = orderID
	req.Ciphertext = ciphertext
	return req, nil
}

// DecodeCancelOrder decodes abi.encode(uint256 orderId).
func DecodeCancelOrder(data []byte) (CancelOrderRequest, error) {
	var req CancelOrderRequest
	values, err := unpackStrict(CancelOrderMessageArgs, data)
	if err != nil {
		return req, fmt.Errorf("unpacking cancel order message: %w", err)
	}
	orderID, ok := values[0].(*big.Int)
	if !ok {
		return req, fmt.Errorf("orderId: expected *big.Int, got %T", values[0])
	}
	req.OrderID = orderID
	return req, nil
}

// unpackStrict unpacks data with args and rejects sloppy encodings by
// requiring that re-packing the decoded values reproduces the input
// byte-for-byte (same guarantee structs.Decode gives for single arguments).
func unpackStrict(args abi.Arguments, data []byte) ([]any, error) {
	values, err := args.Unpack(data)
	if err != nil {
		return nil, err
	}
	repacked, err := args.Pack(values...)
	if err != nil {
		return nil, fmt.Errorf("repacking decoded values: %w", err)
	}
	if len(repacked) != len(data) {
		return nil, fmt.Errorf("non-canonical encoding: %d bytes decoded, %d bytes re-encoded", len(data), len(repacked))
	}
	for i := range repacked {
		if repacked[i] != data[i] {
			return nil, fmt.Errorf("non-canonical encoding: mismatch at byte %d", i)
		}
	}
	return values, nil
}

// OrderState is one order as exposed via GET /state. It deliberately contains
// ONLY the order id and status — trigger prices never leave the enclave.
type OrderState struct {
	OrderID string `json:"orderId"`
	Status  string `json:"status"`
}

// State holds the extension's observable state, returned by GET /state.
type State struct {
	EncryptionPubKey string       `json:"encryptionPubKey"`
	OpenOrders       int          `json:"openOrders"`
	Orders           []OrderState `json:"orders"`
}

// --- DO NOT MODIFY below this line. ---

// StateResponse is the envelope returned by GET /state.
type StateResponse struct {
	StateVersion common.Hash `json:"stateVersion"`
	State        State       `json:"state"`
}
