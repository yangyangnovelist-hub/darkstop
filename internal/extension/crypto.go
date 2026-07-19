package extension

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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

// triggerPlaintext is the decrypted tagged policy. Legacy fixed-price
// plaintexts without strategy remain readable for deployed 0.1.0 clients.
type triggerPlaintext struct {
	Strategy     string  `json:"strategy,omitempty"`
	TriggerPrice *string `json:"triggerPrice,omitempty"`
	TrailBps     *uint16 `json:"trailBps,omitempty"`
}

type OrderPolicy struct {
	TriggerPrice *big.Int
	TrailBps     uint16
}

func ParseOrderPlaintext(plaintext []byte) (OrderPolicy, error) {
	if len(plaintext) == 0 || len(plaintext) > 1024 {
		return OrderPolicy{}, fmt.Errorf("order plaintext must be between 1 and 1024 bytes")
	}
	if err := validateOrderJSONKeys(plaintext); err != nil {
		return OrderPolicy{}, err
	}
	dec := json.NewDecoder(bytes.NewReader(plaintext))
	dec.DisallowUnknownFields()
	var p triggerPlaintext
	if err := dec.Decode(&p); err != nil {
		return OrderPolicy{}, fmt.Errorf("decoding trigger plaintext: %w", err)
	}
	if err := ensureJSONEOF(dec); err != nil {
		return OrderPolicy{}, err
	}

	// Legacy fixed payloads omitted strategy; keep those compatible.
	strategy := p.Strategy
	if strategy == "" && p.TriggerPrice != nil && p.TrailBps == nil {
		strategy = "fixed"
	}
	switch strategy {
	case "trailing":
		if p.TriggerPrice != nil || p.TrailBps == nil {
			return OrderPolicy{}, fmt.Errorf("trailing policy requires only trailBps")
		}
		if *p.TrailBps < 25 || *p.TrailBps > 5000 {
			return OrderPolicy{}, fmt.Errorf("trailBps must be between 25 and 5000")
		}
		return OrderPolicy{TrailBps: *p.TrailBps}, nil
	case "fixed":
		if p.TriggerPrice == nil || p.TrailBps != nil || *p.TriggerPrice == "" {
			return OrderPolicy{}, fmt.Errorf("fixed policy requires only triggerPrice")
		}
		trigger, ok := new(big.Int).SetString(*p.TriggerPrice, 10)
		if !ok || trigger.Sign() <= 0 || trigger.BitLen() > 256 {
			return OrderPolicy{}, fmt.Errorf("triggerPrice must be a positive uint256 base-10 integer")
		}
		return OrderPolicy{TriggerPrice: trigger}, nil
	default:
		return OrderPolicy{}, fmt.Errorf("strategy must be fixed or trailing")
	}
}

// validateOrderJSONKeys rejects duplicate and case-variant keys before the
// standard decoder can silently keep the last value. Order policies are flat,
// so exact top-level keys make the signed plaintext unambiguous.
func validateOrderJSONKeys(plaintext []byte) error {
	dec := json.NewDecoder(bytes.NewReader(plaintext))
	first, err := dec.Token()
	if err != nil {
		return fmt.Errorf("decoding trigger plaintext: %w", err)
	}
	if delim, ok := first.(json.Delim); !ok || delim != '{' {
		return fmt.Errorf("order plaintext must be a JSON object")
	}
	allowed := map[string]bool{"strategy": true, "triggerPrice": true, "trailBps": true}
	seen := make(map[string]bool, len(allowed))
	for dec.More() {
		token, err := dec.Token()
		if err != nil {
			return fmt.Errorf("decoding order key: %w", err)
		}
		key, ok := token.(string)
		if !ok {
			return fmt.Errorf("order plaintext contains a non-string key")
		}
		if !allowed[key] {
			return fmt.Errorf("unknown order field %q", key)
		}
		if seen[key] {
			return fmt.Errorf("duplicate order field %q", key)
		}
		seen[key] = true
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return fmt.Errorf("decoding order field %q: %w", key, err)
		}
	}
	if _, err := dec.Token(); err != nil {
		return fmt.Errorf("decoding order object end: %w", err)
	}
	if err := ensureJSONEOF(dec); err != nil {
		return err
	}
	return nil
}

func ensureJSONEOF(dec *json.Decoder) error {
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("order plaintext contains trailing JSON")
		}
		return fmt.Errorf("decoding trailing plaintext: %w", err)
	}
	return nil
}

// ParseTriggerPlaintext parses and validates a decrypted order payload,
// returning the trigger price (USD per FLR, scaled to 6 decimals) as a
// positive integer.
func ParseTriggerPlaintext(plaintext []byte) (*big.Int, error) {
	policy, err := ParseOrderPlaintext(plaintext)
	if err != nil {
		return nil, err
	}
	if policy.TriggerPrice == nil {
		return nil, fmt.Errorf("payload is a trailing-stop policy")
	}
	return policy.TriggerPrice, nil
}
