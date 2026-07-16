//go:generate go run github.com/ethereum/go-ethereum/cmd/abigen --abi=DarkStopVault.abi --bin=DarkStopVault.bin --pkg=darkstop --type=DarkStopVault --out=autogen.go
//go:generate go run github.com/ethereum/go-ethereum/cmd/abigen --abi=MockUSDT0.abi --bin=MockUSDT0.bin --pkg=darkstop --type=MockUSDT0 --out=autogen_usdt0.go

package darkstop
