package utils

import (
	"context"
	"math/big"
	"strings"
	"time"

	"extension-scaffold/tools/pkg/contracts/darkstop"
	"extension-scaffold/tools/pkg/fccutils"
	"extension-scaffold/tools/pkg/support"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

// FlareContractRegistry is deployed at the same address on every Flare
// network (flare, songbird, coston, coston2).
var FlareContractRegistry = common.HexToAddress("0xaD67FE66660Fb8dFE9d6b1b4240d8650e30F6019")

// DefaultInstructionFee is the native fee (wei) forwarded to the TEE
// extension registry per instruction. The registry enforces a minimum
// (FeeTooLow); overpaying is accepted. Same convention as the scaffold.
var DefaultInstructionFee = big.NewInt(1000000)

const contractRegistryABI = `[{"inputs":[{"internalType":"string","name":"_name","type":"string"}],"name":"getContractAddressByName","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}]`

// ResolveFtsoV2 reads the FtsoV2 address from the on-chain
// FlareContractRegistry. This mirrors what the periphery library's
// ContractRegistry.getTestFtsoV2() does at contract runtime (both resolve
// the name "FtsoV2").
func ResolveFtsoV2(s *support.Support) (common.Address, error) {
	parsed, err := abi.JSON(strings.NewReader(contractRegistryABI))
	if err != nil {
		return common.Address{}, errors.Errorf("parsing registry ABI: %s", err)
	}
	callData, err := parsed.Pack("getContractAddressByName", "FtsoV2")
	if err != nil {
		return common.Address{}, errors.Errorf("packing registry call: %s", err)
	}
	res, err := s.ChainClient.CallContract(context.Background(), ethereum.CallMsg{
		To:   &FlareContractRegistry,
		Data: callData,
	}, nil)
	if err != nil {
		return common.Address{}, errors.Errorf("calling FlareContractRegistry: %s", err)
	}
	values, err := parsed.Unpack("getContractAddressByName", res)
	if err != nil {
		return common.Address{}, errors.Errorf("unpacking registry result: %s", err)
	}
	addr, ok := values[0].(common.Address)
	if !ok {
		return common.Address{}, errors.Errorf("registry result: expected address, got %T", values[0])
	}
	if addr == (common.Address{}) {
		return common.Address{}, errors.New("FlareContractRegistry returned zero address for FtsoV2")
	}
	return addr, nil
}

// DeployMockUSDT0 deploys the 6-decimals testnet payout token.
func DeployMockUSDT0(s *support.Support) (common.Address, *darkstop.MockUSDT0, error) {
	opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
	if err != nil {
		return common.Address{}, nil, errors.Errorf("failed to create transactor: %s", err)
	}

	address, tx, contract, err := darkstop.DeployMockUSDT0(opts, s.ChainClient)
	if err != nil {
		return common.Address{}, nil, errors.Errorf("failed to deploy MockUSDT0: %s", err)
	}

	if err := waitMinedOK(s, tx, "MockUSDT0 deployment"); err != nil {
		return common.Address{}, nil, err
	}
	return address, contract, nil
}

// DeployVault deploys DarkStopVault. Both registry args are the
// FlareTeeManager diamond proxy: the diamond routes ExtensionManager and
// MachineManager calls to the right facets.
func DeployVault(
	s *support.Support, ftsoV2, payoutToken common.Address, instructionFee *big.Int,
) (common.Address, *darkstop.DarkStopVault, error) {
	opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
	if err != nil {
		return common.Address{}, nil, errors.Errorf("failed to create transactor: %s", err)
	}

	address, tx, contract, err := darkstop.DeployDarkStopVault(
		opts, s.ChainClient,
		s.Addresses.FlareTeeManager, s.Addresses.FlareTeeManager,
		ftsoV2, payoutToken, instructionFee,
	)
	if err != nil {
		return common.Address{}, nil, errors.Errorf("failed to deploy DarkStopVault: %s", err)
	}

	if err := waitMinedOK(s, tx, "DarkStopVault deployment"); err != nil {
		return common.Address{}, nil, err
	}
	return address, contract, nil
}

// MintPayoutPool mints `amount` MockUSDT0 base units to the vault so
// settlements can pay out. Owner (deployer) only.
func MintPayoutPool(s *support.Support, token *darkstop.MockUSDT0, vault common.Address, amount *big.Int) error {
	opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
	if err != nil {
		return errors.Errorf("failed to create transactor: %s", err)
	}
	tx, err := token.Mint(opts, vault, amount)
	if err != nil {
		return errors.Errorf("failed to mint payout pool: %s", err)
	}
	return waitMinedOK(s, tx, "payout pool mint")
}

// SetTeeExecutor sets the address allowed to call settle(). Owner only.
func SetTeeExecutor(s *support.Support, vault *darkstop.DarkStopVault, executor common.Address) error {
	opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
	if err != nil {
		return errors.Errorf("failed to create transactor: %s", err)
	}
	tx, err := vault.SetTeeExecutor(opts, executor)
	if err != nil {
		return errors.Errorf("failed to call setTeeExecutor: %s", err)
	}
	return waitMinedOK(s, tx, "setTeeExecutor")
}

func SetExtensionId(s *support.Support, vaultAddress common.Address) error {
	vault, err := darkstop.NewDarkStopVault(vaultAddress, s.ChainClient)
	if err != nil {
		return errors.Errorf("failed to bind contract: %s", err)
	}

	opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
	if err != nil {
		return errors.Errorf("failed to create transactor: %s", err)
	}

	tx, err := vault.SetExtensionId(opts)
	if err != nil {
		reason := fccutils.DecodeRevertReason(err)
		if reason == "" {
			reason = simulateVaultCall(s, vaultAddress, nil, "setExtensionId")
		}
		if reason != "" {
			return errors.Errorf("failed to call setExtensionId: %s (revert reason: %s)", err, reason)
		}
		return errors.Errorf("failed to call setExtensionId: %s", err)
	}

	receipt, err := bind.WaitMined(context.Background(), s.ChainClient, tx)
	if err != nil {
		return errors.Errorf("failed waiting for transaction: %s", err)
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		reason := simulateVaultCall(s, vaultAddress, nil, "setExtensionId")
		if reason != "" {
			return errors.Errorf("setExtensionId transaction failed (revert reason: %s)", reason)
		}
		return errors.New("setExtensionId transaction failed")
	}

	return nil
}

// SendPlaceOrder calls placeOrder(ciphertext) with `value` attached
// (value = instruction fee + deposit). Returns the FCC instruction id, the
// new order id and the tx hash.
func SendPlaceOrder(
	s *support.Support, vaultAddress common.Address, ciphertext []byte, value *big.Int,
) (common.Hash, *big.Int, common.Hash, error) {
	vault, err := darkstop.NewDarkStopVault(vaultAddress, s.ChainClient)
	if err != nil {
		return common.Hash{}, nil, common.Hash{}, errors.Errorf("failed to bind contract: %s", err)
	}

	opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
	if err != nil {
		return common.Hash{}, nil, common.Hash{}, errors.Errorf("failed to create transactor: %s", err)
	}
	opts.Value = value

	tx, err := vault.PlaceOrder(opts, ciphertext)
	if err != nil {
		reason := fccutils.DecodeRevertReason(err)
		if reason == "" {
			reason = simulateVaultCall(s, vaultAddress, value, "placeOrder", ciphertext)
		}
		if reason != "" {
			return common.Hash{}, nil, common.Hash{}, errors.Errorf("failed to place order: %s (revert reason: %s)", err, reason)
		}
		return common.Hash{}, nil, common.Hash{}, errors.Errorf("failed to place order: %s", err)
	}

	receipt, err := bind.WaitMined(context.Background(), s.ChainClient, tx)
	if err != nil {
		return common.Hash{}, nil, common.Hash{}, errors.Errorf("failed waiting for transaction: %s", err)
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		reason := simulateVaultCall(s, vaultAddress, value, "placeOrder", ciphertext)
		if reason != "" {
			return common.Hash{}, nil, common.Hash{}, errors.Errorf("placeOrder transaction failed (revert reason: %s)", reason)
		}
		return common.Hash{}, nil, common.Hash{}, errors.Errorf("placeOrder transaction failed with status: %d", receipt.Status)
	}

	// The receipt carries the vault's OrderPlaced event plus the registry's
	// TeeInstructionsSent event — scan for both.
	var orderID *big.Int
	var instructionID common.Hash
	for _, lg := range receipt.Logs {
		if placed, err := vault.ParseOrderPlaced(*lg); err == nil {
			orderID = placed.OrderId
			continue
		}
		if sent, err := s.TeeVerification.ParseTeeInstructionsSent(*lg); err == nil {
			instructionID = sent.InstructionId
		}
	}
	if orderID == nil {
		return common.Hash{}, nil, common.Hash{}, errors.New("OrderPlaced event not found in receipt")
	}
	if instructionID == (common.Hash{}) {
		return common.Hash{}, nil, common.Hash{}, errors.New("TeeInstructionsSent event not found in receipt")
	}

	return instructionID, orderID, receipt.TxHash, nil
}

// simulateVaultCall re-simulates a vault call to extract a revert reason.
// Returns "" if no reason could be recovered.
func simulateVaultCall(s *support.Support, vaultAddress common.Address, value *big.Int, method string, args ...interface{}) string {
	parsed, _ := darkstop.DarkStopVaultMetaData.GetAbi()
	if parsed == nil {
		return ""
	}
	callData, packErr := parsed.Pack(method, args...)
	if packErr != nil {
		return ""
	}
	from := crypto.PubkeyToAddress(s.Prv.PublicKey)
	return fccutils.SimulateAndDecodeRevert(s.ChainClient, from, vaultAddress, value, callData)
}

// waitMinedOK waits for tx to be mined and fails on a reverted receipt.
func waitMinedOK(s *support.Support, tx *types.Transaction, what string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	receipt, err := bind.WaitMined(ctx, s.ChainClient, tx)
	if err != nil {
		return errors.Errorf("%s tx not mined within 2 minutes (tx: %s): %s", what, tx.Hash().Hex(), err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return errors.Errorf("%s transaction failed (tx: %s)", what, tx.Hash().Hex())
	}
	return nil
}
