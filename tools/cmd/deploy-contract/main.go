// deploy-contract deploys the DarkStop stack:
//
//  1. MockUSDT0 (testnet payout token)
//  2. DarkStopVault (FTSO address resolved on-chain via FlareContractRegistry)
//  3. Mints a 1,000,000 USDT0 payout pool (6 decimals) to the vault
//  4. setTeeExecutor(deployer) — testnet simplification: the deployer key is
//     also the watcher's settlement key
//
// The last stdout line is the vault address (consumed by pre-build.sh).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"extension-scaffold/tools/pkg/configs"
	"extension-scaffold/tools/pkg/fccutils"
	"extension-scaffold/tools/pkg/support"
	instrutils "extension-scaffold/tools/pkg/utils"
	"extension-scaffold/tools/pkg/validate"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
)

// payoutPool is 1,000,000 USDT0 in 6-decimals base units.
var payoutPool = new(big.Int).Mul(big.NewInt(1_000_000), big.NewInt(1_000_000))

type deployInfo struct {
	MockUSDT0     string `json:"mockUSDT0"`
	DarkStopVault string `json:"darkStopVault"`
	FtsoV2        string `json:"ftsoV2"`
	TeeExecutor   string `json:"teeExecutor"`
	InstructionFee string `json:"instructionFee"`
	PayoutPool    string `json:"payoutPool"`
}

func main() {
	af := flag.String("a", configs.AddressesFile, "file with deployed addresses")
	cf := flag.String("c", configs.ChainNodeURL, "chain node url")
	outFile := flag.String("o", "", "write deployed vault address to this file (optional)")
	infoFile := flag.String("deployInfo", "", "write full deployment info JSON to this file (optional)")
	feeF := flag.Int64("fee", instrutils.DefaultInstructionFee.Int64(), "instruction fee in wei forwarded to the registry per instruction")
	preflightOnly := flag.Bool("preflight-only", false, "run validation checks and exit without deploying")
	flag.Parse()

	testSupport, err := support.DefaultSupport(*af, *cf)
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	// --- Pre-flight validation ---
	deployer := crypto.PubkeyToAddress(testSupport.Prv.PublicKey)
	logger.Infof("Deployer:             %s", deployer.Hex())
	logger.Infof("Chain ID:             %s", testSupport.ChainID.String())
	logger.Infof("FlareTeeManager:      %s", testSupport.Addresses.FlareTeeManager.Hex())

	if err := validate.AddressNotZero(testSupport.Addresses.FlareTeeManager, "FlareTeeManager"); err != nil {
		fccutils.FatalWithCause(err)
	}
	if err := validate.AddressHasCode(testSupport.ChainClient, testSupport.Addresses.FlareTeeManager, "FlareTeeManager"); err != nil {
		fccutils.FatalWithCause(err)
	}
	if err := validate.KeyHasFunds(testSupport.ChainClient, testSupport.Prv, validate.MinDeployBalance); err != nil {
		fccutils.FatalWithCause(err)
	}

	// Resolve FtsoV2 from the on-chain FlareContractRegistry (what the
	// periphery ContractRegistry.getTestFtsoV2() resolves at runtime).
	ftsoV2, err := instrutils.ResolveFtsoV2(testSupport)
	if err != nil {
		fccutils.FatalWithCause(err)
	}
	logger.Infof("FtsoV2 (registry):    %s", ftsoV2.Hex())
	if err := validate.AddressHasCode(testSupport.ChainClient, ftsoV2, "FtsoV2"); err != nil {
		fccutils.FatalWithCause(err)
	}

	if *preflightOnly {
		logger.Infof("Pre-flight checks passed. Exiting without deploying.")
		return
	}

	instructionFee := big.NewInt(*feeF)
	logger.Infof("Instruction fee:      %s wei", instructionFee.String())

	// --- 1. MockUSDT0 ---
	logger.Infof("Deploying MockUSDT0...")
	usdt0Addr, usdt0, err := instrutils.DeployMockUSDT0(testSupport)
	if err != nil {
		fccutils.FatalWithCause(err)
	}
	logger.Infof("MockUSDT0 deployed at: %s", usdt0Addr.Hex())

	// --- 2. DarkStopVault ---
	logger.Infof("Deploying DarkStopVault...")
	vaultAddr, vault, err := instrutils.DeployVault(testSupport, ftsoV2, usdt0Addr, instructionFee)
	if err != nil {
		fccutils.FatalWithCause(err)
	}
	logger.Infof("DarkStopVault deployed at: %s", vaultAddr.Hex())

	// --- 3. Payout pool ---
	logger.Infof("Minting payout pool (%s base units USDT0) to vault...", payoutPool.String())
	if err := instrutils.MintPayoutPool(testSupport, usdt0, vaultAddr, payoutPool); err != nil {
		fccutils.FatalWithCause(err)
	}
	logger.Infof("Payout pool minted")

	// --- 4. TEE executor (testnet: deployer settles) ---
	logger.Infof("Setting teeExecutor to deployer %s...", deployer.Hex())
	if err := instrutils.SetTeeExecutor(testSupport, vault, deployer); err != nil {
		fccutils.FatalWithCause(err)
	}
	logger.Infof("teeExecutor set")

	// Optionally write address / full info for script consumption
	if *outFile != "" {
		os.MkdirAll(filepath.Dir(*outFile), 0755)
		os.WriteFile(*outFile, []byte(vaultAddr.Hex()), 0644)
	}
	if *infoFile != "" {
		info := deployInfo{
			MockUSDT0:      usdt0Addr.Hex(),
			DarkStopVault:  vaultAddr.Hex(),
			FtsoV2:         ftsoV2.Hex(),
			TeeExecutor:    deployer.Hex(),
			InstructionFee: instructionFee.String(),
			PayoutPool:     payoutPool.String(),
		}
		data, _ := json.MarshalIndent(info, "", "  ")
		os.MkdirAll(filepath.Dir(*infoFile), 0755)
		if err := os.WriteFile(*infoFile, append(data, '\n'), 0644); err != nil {
			fccutils.FatalWithCause(err)
		}
		logger.Infof("Deployment info written to %s", *infoFile)
	}

	// Machine-readable output on stdout (for scripts): vault address last.
	fmt.Println(vaultAddr.Hex())
}
