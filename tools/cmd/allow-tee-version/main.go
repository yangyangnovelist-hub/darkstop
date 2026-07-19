package main

import (
	"crypto/ecdsa"
	"extension-scaffold/tools/pkg/configs"
	"extension-scaffold/tools/pkg/fccutils"
	"extension-scaffold/tools/pkg/support"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"os"
	"strings"
)

func main() {
	af := flag.String("a", configs.AddressesFile, "file with deployed addresses")
	cf := flag.String("c", configs.ChainNodeURL, "chain node url")
	pf := flag.String("p", configs.ExtensionProxyURL, "proxy url")
	versionF := flag.String("version", "v0.2.0", "version")
	flag.Parse()

	testSupport, err := support.DefaultSupport(*af, *cf)
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	// get teeID from proxy
	teeInfo, err := fccutils.TeeInfo(*pf)
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	var privKey *ecdsa.PrivateKey
	privKeyString := os.Getenv("EXTENSION_OWNER_KEY")
	if privKeyString != "" {
		if strings.HasPrefix(privKeyString, "0x") || strings.HasPrefix(privKeyString, "0X") {
			privKeyString = privKeyString[2:]
		}
		privKey, err = crypto.HexToECDSA(privKeyString)
		if err != nil {
			fccutils.FatalWithCause(err)
		}
	} else {
		privKey = testSupport.Prv
	}

	keySource := "EXTENSION_OWNER_KEY"
	if privKeyString == "" {
		keySource = "DEPLOYMENT_PRIVATE_KEY (default)"
	}
	logger.Infof("Using key: %s (deployer: %s)", keySource, crypto.PubkeyToAddress(privKey.PublicKey).Hex())

	teeID, _, err := fccutils.TeeProxyId(teeInfo)
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	logger.Infof("Code hash:    %s (source: proxy /info)", teeInfo.MachineData.CodeHash.Hex())
	logger.Infof("Platform:     %s (source: proxy /info)", teeInfo.MachineData.Platform.Hex())
	logger.Infof("Extension ID: %s", teeInfo.MachineData.ExtensionID.Big().String())
	logger.Infof("TEE ID:       %s", teeID.Hex())
	logger.Infof("Version:      %s", *versionF)
	logger.Warnf("NOTE: Code hash is from proxy /info response — not independently verified against attestation")

	// Idempotency: skip if this codeHash+platform combo is already registered and active.
	// Avoids sending a tx that would revert with VersionAlreadyExists() on re-runs.
	supported, err := testSupport.TeeExtensionRegistry.IsCodeHashPlatformSupported(
		nil,
		teeInfo.MachineData.ExtensionID.Big(),
		teeInfo.MachineData.CodeHash,
		teeInfo.MachineData.Platform,
	)
	if err != nil {
		fccutils.FatalWithCause(err)
	}
	if supported {
		logger.Infof("version already registered for this code hash + platform, skipping")
		return
	}

	err = fccutils.AddTeeVersion(testSupport, privKey, teeInfo.MachineData.ExtensionID.Big(), teeInfo.MachineData.CodeHash, teeInfo.MachineData.Platform, common.Hash{}, *versionF)
	if err != nil {
		if strings.Contains(err.Error(), "VersionAlreadyExists") {
			// A version-name collision is safe to skip only if this exact image
			// is already supported. Otherwise swallowing the error would leave a
			// new code hash undeployable while reporting success.
			supported, checkErr := testSupport.TeeExtensionRegistry.IsCodeHashPlatformSupported(
				nil,
				teeInfo.MachineData.ExtensionID.Big(),
				teeInfo.MachineData.CodeHash,
				teeInfo.MachineData.Platform,
			)
			if checkErr != nil {
				fccutils.FatalWithCause(checkErr)
			}
			if !supported {
				fccutils.FatalWithCause(fmt.Errorf("version %s already exists but does not support code hash %s on platform %s: %w", *versionF, teeInfo.MachineData.CodeHash.Hex(), teeInfo.MachineData.Platform.Hex(), err))
			}
			logger.Infof("version already registered for this code hash + platform, skipping")
		} else {
			fccutils.FatalWithCause(err)
		}
	}
}
