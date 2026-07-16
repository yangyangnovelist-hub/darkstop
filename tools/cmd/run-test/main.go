// run-test sends a PLACE_ORDER instruction end-to-end on a live chain:
//
//  1. Fetches the extension's ECIES public key from GET <proxy>/state
//  2. Encrypts {"triggerPrice": ...} to that key (go-ethereum ecies —
//     same ECIES_AES128_SHA256 format the extension decrypts)
//  3. Calls DarkStopVault.placeOrder(ciphertext) with fee + deposit attached
//  4. Polls the extension proxy and asserts the result is
//     {"orderId": "<id>", "status": "open"}
//
// The trigger price never appears on-chain: only the ciphertext does.
package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"extension-scaffold/pkg/types"
	"extension-scaffold/tools/pkg/configs"
	"extension-scaffold/tools/pkg/fccutils"
	"extension-scaffold/tools/pkg/support"
	instrutils "extension-scaffold/tools/pkg/utils"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/pkg/errors"

	"extension-scaffold/tools/pkg/contracts/darkstop"
)

func main() {
	af := flag.String("a", configs.AddressesFile, "file with deployed addresses")
	cf := flag.String("c", configs.ChainNodeURL, "chain node url")
	pf := flag.String("p", configs.ExtensionProxyURL, "extension proxy url")
	vaultF := flag.String("instructionSender", "", "DarkStopVault address")
	depositF := flag.String("deposit", "50000000000000000", "order deposit in wei (default 0.05 C2FLR)")
	triggerF := flag.String("trigger", "1000000000", "trigger price, USD/FLR with 6 decimals (default $1000 — far above spot, instant trigger)")
	flag.Parse()

	vaultAddress := common.HexToAddress(*vaultF)
	if vaultAddress == (common.Address{}) {
		logger.Fatal("--instructionSender (vault address) is required")
	}

	deposit, ok := new(big.Int).SetString(*depositF, 10)
	if !ok || deposit.Sign() <= 0 {
		logger.Fatalf("invalid -deposit: %q", *depositF)
	}
	trigger, ok := new(big.Int).SetString(*triggerF, 10)
	if !ok || trigger.Sign() <= 0 {
		logger.Fatalf("invalid -trigger: %q", *triggerF)
	}

	testSupport, err := support.DefaultSupport(*af, *cf)
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	// --- Generic: configure contract -----------------------------------------
	logger.Infof("Setting extension ID on vault...")
	err = instrutils.SetExtensionId(testSupport, vaultAddress)
	if err != nil {
		if strings.Contains(err.Error(), "already set") || strings.Contains(err.Error(), "Extension ID already set") {
			logger.Infof("Extension ID already set on contract, continuing")
		} else {
			logger.Errorf("setExtensionId failed: %s", err)
			fccutils.FatalWithCause(errors.Errorf(
				"setExtensionId failed — is the extension registered? Check that pre-build.sh completed successfully. Error: %s", err))
		}
	}

	// --- Fetch the enclave's encryption pubkey -------------------------------
	logger.Infof("Fetching extension state from %s/state ...", *pf)
	pubKey, err := fetchEncryptionKey(*pf)
	if err != nil {
		fccutils.FatalWithCause(err)
	}
	logger.Infof("Extension encryption pubkey: %s...", pubKeyPreview(pubKey))

	// --- Encrypt the trigger price client-side -------------------------------
	plaintext, err := json.Marshal(map[string]string{"triggerPrice": trigger.String()})
	if err != nil {
		fccutils.FatalWithCause(err)
	}
	ciphertext, err := encryptToExtension(pubKey, plaintext)
	if err != nil {
		fccutils.FatalWithCause(err)
	}
	logger.Infof("Ciphertext: %d bytes (trigger price stays off-chain)", len(ciphertext))

	// --- Read the on-chain instruction fee and place the order ---------------
	vault, err := darkstop.NewDarkStopVault(vaultAddress, testSupport.ChainClient)
	if err != nil {
		fccutils.FatalWithCause(err)
	}
	fee, err := vault.INSTRUCTIONFEE(&bind.CallOpts{})
	if err != nil {
		fccutils.FatalWithCause(errors.Errorf("reading INSTRUCTION_FEE: %s", err))
	}
	value := new(big.Int).Add(fee, deposit)

	logger.Infof("Sending PLACE_ORDER (deposit %s wei + fee %s wei)...", deposit.String(), fee.String())
	instructionID, orderID, txHash, err := instrutils.SendPlaceOrder(testSupport, vaultAddress, ciphertext, value)
	if err != nil {
		fccutils.FatalWithCause(err)
	}
	logger.Infof("Order %s placed. Tx: %s", orderID.String(), txHash.Hex())
	logger.Infof("Instruction ID: %s", instructionID.Hex())

	time.Sleep(5 * time.Second)

	// --- Verify the enclave accepted the order --------------------------------
	if err := verifyPlaceOrderResult(*pf, instructionID, orderID); err != nil {
		fccutils.FatalWithCause(err)
	}
	logger.Infof("Test passed: PLACE_ORDER processed, order %s is open in the enclave", orderID.String())
	logger.Infof("All tests passed.")
}

// fetchEncryptionKey GETs <proxyURL>/state and returns the extension's ECIES
// public key.
func fetchEncryptionKey(proxyURL string) (*ecies.PublicKey, error) {
	resp, err := http.Get(strings.TrimRight(proxyURL, "/") + "/state")
	if err != nil {
		return nil, errors.Errorf("GET /state: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("GET /state: unexpected status %d", resp.StatusCode)
	}

	var sr types.StateResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, errors.Errorf("decoding state response: %s", err)
	}
	if sr.State.EncryptionPubKey == "" {
		return nil, errors.New("state response has empty encryptionPubKey")
	}

	raw, err := hexutil.Decode(sr.State.EncryptionPubKey)
	if err != nil {
		return nil, errors.Errorf("decoding encryptionPubKey hex: %s", err)
	}
	ecdsaPub, err := ethcrypto.UnmarshalPubkey(raw)
	if err != nil {
		return nil, errors.Errorf("parsing encryptionPubKey: %s", err)
	}
	return ecies.ImportECDSAPublic(ecdsaPub), nil
}

// encryptToExtension ECIES-encrypts plaintext to the extension's public key
// using go-ethereum's ecies (ECIES_AES128_SHA256 for secp256k1) — the exact
// format internal/extension/crypto.go decrypts (see testdata/ecies_vector.json).
func encryptToExtension(pub *ecies.PublicKey, plaintext []byte) ([]byte, error) {
	ciphertext, err := ecies.Encrypt(rand.Reader, pub, plaintext, nil, nil)
	if err != nil {
		return nil, errors.Errorf("ecies encrypt: %s", err)
	}
	return ciphertext, nil
}

func verifyPlaceOrderResult(proxyURL string, instructionID common.Hash, orderID *big.Int) error {
	// --- Generic: poll proxy for result (do not modify) ---
	actionResponse, err := fccutils.ActionResult(proxyURL, instructionID)
	if err != nil {
		return err
	}
	actionResult := actionResponse.Result

	if actionResult.Status == 0 {
		return errors.Errorf("instruction processing failed: %s", actionResult.Log)
	}
	if actionResult.Status == 2 {
		return errors.New("instruction still pending after polling, expected completed")
	}

	if len(actionResult.Data) == 0 {
		return errors.New("expected response data but got none")
	}

	var resp types.OrderResponse
	if err := json.Unmarshal(actionResult.Data, &resp); err != nil {
		return errors.Errorf("failed to unmarshal response: %s", err)
	}

	if resp.OrderID != orderID.String() {
		return errors.Errorf("expected orderId %q, got %q", orderID.String(), resp.OrderID)
	}
	if resp.Status != "open" {
		return errors.Errorf(`expected status "open", got %q`, resp.Status)
	}

	logger.Infof("Response data: %+v", resp)

	return nil
}

func pubKeyPreview(pub *ecies.PublicKey) string {
	raw := ethcrypto.FromECDSAPub(pub.ExportECDSA())
	s := fmt.Sprintf("0x%x", raw)
	if len(s) > 20 {
		return s[:20]
	}
	return s
}
