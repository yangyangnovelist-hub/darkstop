package extension

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"

	"extension-scaffold/internal/config"
	"extension-scaffold/pkg/types"

	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/tee/instruction"
	teetypes "github.com/flare-foundation/tee-node/pkg/types"
	teeutils "github.com/flare-foundation/tee-node/pkg/utils"

	"github.com/flare-foundation/tee-node/pkg/processorutils"
)

type Extension struct {
	Server *http.Server

	// crypto is the enclave's ECIES identity: clients encrypt trigger
	// parameters to its public key (served via GET /state).
	crypto *Crypto
	// store holds decrypted orders in enclave memory only.
	store *Store
}

// New constructs the extension: fresh enclave keypair, empty order store,
// HTTP routes. It panics if key generation fails (extension cannot operate
// without its decryption identity).
func New(extensionPort, signPort int) *Extension {
	var c *Crypto
	var err error
	if config.EnclavePrivateKey != "" {
		c, err = NewCryptoFromHex(config.EnclavePrivateKey)
	} else {
		c, err = NewCrypto()
	}
	if err != nil {
		panic(fmt.Sprintf("initializing enclave keypair: %v", err))
	}

	e := &Extension{crypto: c, store: NewStore()}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /state", e.stateHandler)
	mux.HandleFunc("POST /action", e.actionHandler)

	e.Server = &http.Server{Addr: fmt.Sprintf(":%d", extensionPort), Handler: mux}
	return e
}

// Store exposes the order store for the FTSO watcher.
func (e *Extension) Store() *Store {
	return e.store
}

// stateHandler serves the enclave's public key and the order list.
// OrderState carries ONLY orderId and status — never trigger prices.
func (e *Extension) stateHandler(w http.ResponseWriter, r *http.Request) {
	orders := e.store.Snapshot()
	orderStates := make([]types.OrderState, 0, len(orders))
	open := 0
	for _, o := range orders {
		if o.Status == StatusOpen {
			open++
		}
		orderStates = append(orderStates, types.OrderState{
			OrderID: strconv.FormatUint(o.ID, 10),
			Status:  o.Status,
		})
	}

	stateResponse := types.StateResponse{
		StateVersion: teeutils.ToHash(config.Version),
		State: types.State{
			EncryptionPubKey:  e.crypto.PublicKeyHex(),
			SupportedPolicies: e.store.SupportedPolicies(),
			OpenOrders:        open,
			Orders:            orderStates,
		},
	}

	err := json.NewEncoder(w).Encode(stateResponse)
	if err != nil {
		http.Error(w, fmt.Sprintf("sending response: %v", err), http.StatusInternalServerError)
		return
	}
}

func (e *Extension) processAction(action teetypes.Action) (int, []byte) {
	dataFixed, err := processorutils.Parse[instruction.DataFixed](action.Data.Message)
	if err != nil {
		return http.StatusBadRequest, []byte(fmt.Sprintf("decoding fixed data: %v", err))
	}

	switch {
	case dataFixed.OPType == teeutils.ToHash(config.OPTypeDarkstop):
		return e.processDarkstop(action, dataFixed)

	default:
		return http.StatusNotImplemented, []byte(fmt.Sprintf(
			"unsupported op type: received %s, expected %s (%s)",
			dataFixed.OPType.Hex(), teeutils.ToHash(config.OPTypeDarkstop).Hex(), config.OPTypeDarkstop,
		))
	}
}

// processDarkstop routes DARKSTOP instructions by OPCommand.
func (e *Extension) processDarkstop(action teetypes.Action, df *instruction.DataFixed) (int, []byte) {
	switch {
	case df.OPCommand == teeutils.ToHash(config.OPCommandPlaceOrder):
		ar := e.processPlaceOrder(action, df)
		b, _ := json.Marshal(ar)
		return http.StatusOK, b

	case df.OPCommand == teeutils.ToHash(config.OPCommandCancelOrder):
		ar := e.processCancelOrder(action, df)
		b, _ := json.Marshal(ar)
		return http.StatusOK, b

	default:
		return http.StatusNotImplemented, []byte(fmt.Sprintf(
			"unsupported op command: received %s, expected one of [%s (%s), %s (%s)]",
			df.OPCommand.Hex(),
			teeutils.ToHash(config.OPCommandPlaceOrder).Hex(), config.OPCommandPlaceOrder,
			teeutils.ToHash(config.OPCommandCancelOrder).Hex(), config.OPCommandCancelOrder,
		))
	}
}

// processPlaceOrder handles PLACE_ORDER: ABI decode → ECIES decrypt →
// validate policy → store in enclave memory. The result acknowledges the
// order id and status only — private policy state stays inside the TEE.
func (e *Extension) processPlaceOrder(action teetypes.Action, df *instruction.DataFixed) teetypes.ActionResult {
	req, err := types.DecodePlaceOrder(df.OriginalMessage)
	if err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("decoding request: %w", err))
	}

	id, err := orderIDToUint64(req.OrderID)
	if err != nil {
		return buildResult(action, df, nil, 0, err)
	}

	plaintext, err := e.crypto.Decrypt(req.Ciphertext)
	if err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("decrypting ciphertext: %w", err))
	}

	policy, err := ParseOrderPlaintext(plaintext)
	if err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("validating order policy: %w", err))
	}

	if err := e.store.Put(Order{ID: id, TriggerPrice: policy.TriggerPrice, TrailBps: policy.TrailBps, Status: StatusOpen}); err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("storing order: %w", err))
	}

	// Deliberately no trigger value in this log line: the audit trail must
	// stay price-free until settlement reveals the trigger on-chain.
	logger.Infof("order %d placed: policy decrypted and held in enclave memory", id)

	data, _ := json.Marshal(types.OrderResponse{
		OrderID: strconv.FormatUint(id, 10),
		Status:  StatusOpen,
	})
	return buildResult(action, df, data, 1, nil)
}

// processCancelOrder handles CANCEL_ORDER: drops the order from enclave
// memory. The refund already happened on-chain in DarkStopVault.cancel().
func (e *Extension) processCancelOrder(action teetypes.Action, df *instruction.DataFixed) teetypes.ActionResult {
	req, err := types.DecodeCancelOrder(df.OriginalMessage)
	if err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("decoding request: %w", err))
	}

	id, err := orderIDToUint64(req.OrderID)
	if err != nil {
		return buildResult(action, df, nil, 0, err)
	}

	if !e.store.Delete(id) {
		return buildResult(action, df, nil, 0, fmt.Errorf("order %d not found", id))
	}

	logger.Infof("order %d cancelled: dropped from enclave memory", id)

	data, _ := json.Marshal(types.OrderResponse{
		OrderID: strconv.FormatUint(id, 10),
		Status:  StatusCancelled,
	})
	return buildResult(action, df, data, 1, nil)
}

// orderIDToUint64 validates a decoded order id: vault ids start at 1 and the
// enclave store keys by uint64.
func orderIDToUint64(orderID *big.Int) (uint64, error) {
	if orderID == nil || orderID.Sign() <= 0 || !orderID.IsUint64() {
		return 0, fmt.Errorf("invalid order id %s", orderID)
	}
	return orderID.Uint64(), nil
}
