// Package config contains configuration values and defaults used by the extension.
package config

import (
	"os"
	"strconv"
	"time"
)

const (
	Version = "0.2.0"

	// Operation constants — must match the DarkStopVault.sol bytes32
	// constants byte-for-byte (compared via teeutils.ToHash).
	OPTypeDarkstop       = "DARKSTOP"
	OPCommandPlaceOrder  = "PLACE_ORDER"
	OPCommandCancelOrder = "CANCEL_ORDER"

	TimeoutShutdown = 5 * time.Second

	// WatcherInterval is how often the FTSO watcher polls the FLR/USD feed.
	WatcherInterval = 2 * time.Second
)

// Defaults.
var (
	ExtensionPort   = 8080
	SignPort        = 9090
	TypesServerPort = 8100

	// Watcher wiring (no defaults — the watcher only starts when all
	// three are set).
	ChainURL           = ""
	VaultAddress       = ""
	ExecutorPrivateKey = ""
	// Optional deterministic enclave identity for local demos. Production
	// leaves this empty so the extension generates a fresh key in the TEE.
	EnclavePrivateKey = ""
)

// Environment variables override defaults.
func init() {
	ep := os.Getenv("EXTENSION_PORT")
	sp := os.Getenv("SIGN_PORT")
	tp := os.Getenv("TYPES_SERVER_PORT")

	ChainURL = os.Getenv("CHAIN_URL")
	VaultAddress = os.Getenv("VAULT_ADDRESS")
	ExecutorPrivateKey = os.Getenv("EXECUTOR_PRIVATE_KEY")
	EnclavePrivateKey = os.Getenv("ENCLAVE_PRIVATE_KEY")

	if ep != "" {
		if v, err := strconv.Atoi(ep); err == nil {
			ExtensionPort = v
		}
	}
	if sp != "" {
		if v, err := strconv.Atoi(sp); err == nil {
			SignPort = v
		}
	}
	if tp != "" {
		if v, err := strconv.Atoi(tp); err == nil {
			TypesServerPort = v
		}
	}
}
