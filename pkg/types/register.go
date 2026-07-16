package types

import "extension-scaffold/pkg/decoder"

// funcDecoder adapts a typed decode function to the decoder.Decoder interface.
type funcDecoder[T any] func([]byte) (T, error)

func (f funcDecoder[T]) Decode(data []byte) (any, error) { return f(data) }

// RegisterDecoders registers all type decoders for this extension.
// Extension developers: add new registrations here for each OPType/OPCommand.
func RegisterDecoders(r *decoder.Registry) {
	// PLACE_ORDER message: abi.encode(uint256 orderId, bytes ciphertext)
	r.Register(
		decoder.RegistryKey{OPType: "DARKSTOP", OPCommand: "PLACE_ORDER", Kind: decoder.KindMessage},
		funcDecoder[PlaceOrderRequest](DecodePlaceOrder),
	)
	// PLACE_ORDER result (JSON)
	r.Register(
		decoder.RegistryKey{OPType: "DARKSTOP", OPCommand: "PLACE_ORDER", Kind: decoder.KindResult},
		decoder.NewJSONDecoder[OrderResponse](),
	)
	// CANCEL_ORDER message: abi.encode(uint256 orderId)
	r.Register(
		decoder.RegistryKey{OPType: "DARKSTOP", OPCommand: "CANCEL_ORDER", Kind: decoder.KindMessage},
		funcDecoder[CancelOrderRequest](DecodeCancelOrder),
	)
	// CANCEL_ORDER result (JSON)
	r.Register(
		decoder.RegistryKey{OPType: "DARKSTOP", OPCommand: "CANCEL_ORDER", Kind: decoder.KindResult},
		decoder.NewJSONDecoder[OrderResponse](),
	)
}
