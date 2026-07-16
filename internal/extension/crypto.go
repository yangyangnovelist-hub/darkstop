package extension

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
)

// Crypto is the extension's ECIES identity. The private key is generated
// inside the (simulated) TEE at startup and never leaves the process; the
// public key is served via GET /state so clients can encrypt trigger
// parameters to it.
//
// go-ethereum's crypto/ecies maps secp256k1 to ECIES_AES128_SHA256:
// NIST SP 800-56 concat KDF (SHA-256), AES-128-CTR, HMAC-SHA-256 tag.
// Ciphertext layout: 0x04‖ephemeralPubXY(64)‖IV(16)‖ct‖tag(32).
type Crypto struct {
	key *ecies.PrivateKey
}

// NewCrypto generates a fresh secp256k1 ECIES keypair.
func NewCrypto() (*Crypto, error) {
	key, err := ecies.GenerateKey(rand.Reader, ethcrypto.S256(), nil)
	if err != nil {
		return nil, fmt.Errorf("generating ecies keypair: %w", err)
	}
	return &Crypto{key: key}, nil
}

// NewCryptoFromHex imports a secp256k1 private key from a hex string
// (with or without 0x prefix). Used by tests and fixtures.
func NewCryptoFromHex(privHex string) (*Crypto, error) {
	if len(privHex) >= 2 && privHex[:2] == "0x" {
		privHex = privHex[2:]
	}
	ecdsaKey, err := ethcrypto.HexToECDSA(privHex)
	if err != nil {
		return nil, fmt.Errorf("importing private key: %w", err)
	}
	return &Crypto{key: ecies.ImportECDSA(ecdsaKey)}, nil
}

// PublicKeyHex returns the uncompressed secp256k1 public key as 0x04-prefixed hex.
func (c *Crypto) PublicKeyHex() string {
	return "0x" + hex.EncodeToString(ethcrypto.FromECDSAPub(c.key.PublicKey.ExportECDSA()))
}

// PrivateKeyHex returns the private key scalar as 0x-prefixed hex.
// Exposed only for fixture generation — never serve this over HTTP.
func (c *Crypto) PrivateKeyHex() string {
	return "0x" + hex.EncodeToString(ethcrypto.FromECDSA(c.key.ExportECDSA()))
}

// Encrypt encrypts plaintext to this keypair's own public key.
// Production clients encrypt browser-side; this is used for tests and fixtures.
func (c *Crypto) Encrypt(plaintext []byte) ([]byte, error) {
	return ecies.Encrypt(rand.Reader, &c.key.PublicKey, plaintext, nil, nil)
}

// Decrypt decrypts an ECIES ciphertext produced for this keypair's public key.
func (c *Crypto) Decrypt(ciphertext []byte) ([]byte, error) {
	plaintext, err := c.key.Decrypt(ciphertext, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("ecies decrypt: %w", err)
	}
	return plaintext, nil
}

// triggerPlaintext is the decrypted order payload:
// {"triggerPrice":"<USD/FLR price as a positive integer, 6 decimals>"}.
type triggerPlaintext struct {
	TriggerPrice string `json:"triggerPrice"`
}

// ParseTriggerPlaintext parses and validates a decrypted order payload,
// returning the trigger price (USD per FLR, scaled to 6 decimals) as a
// positive integer.
func ParseTriggerPlaintext(plaintext []byte) (*big.Int, error) {
	dec := json.NewDecoder(bytes.NewReader(plaintext))
	dec.DisallowUnknownFields()
	var p triggerPlaintext
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("decoding trigger plaintext: %w", err)
	}
	if p.TriggerPrice == "" {
		return nil, fmt.Errorf("triggerPrice must not be empty")
	}
	trigger, ok := new(big.Int).SetString(p.TriggerPrice, 10)
	if !ok {
		return nil, fmt.Errorf("triggerPrice %q is not a base-10 integer", p.TriggerPrice)
	}
	if trigger.Sign() <= 0 {
		return nil, fmt.Errorf("triggerPrice must be positive, got %s", trigger)
	}
	return trigger, nil
}
