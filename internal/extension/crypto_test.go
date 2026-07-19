package extension

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewCrypto_PublicKeyHex(t *testing.T) {
	c, err := NewCrypto()
	if err != nil {
		t.Fatalf("generating keypair: %v", err)
	}

	pub := c.PublicKeyHex()
	if !strings.HasPrefix(pub, "0x04") {
		t.Errorf("expected uncompressed secp256k1 pubkey (0x04 prefix), got %s", pub)
	}
	// 0x + 65 bytes (uncompressed point) = 2 + 130 chars.
	if len(pub) != 132 {
		t.Errorf("expected 132-char hex pubkey, got %d chars: %s", len(pub), pub)
	}
	if _, err := hex.DecodeString(pub[2:]); err != nil {
		t.Errorf("pubkey is not valid hex: %v", err)
	}
}

func TestNewCrypto_KeysAreFresh(t *testing.T) {
	a, err := NewCrypto()
	if err != nil {
		t.Fatalf("generating keypair: %v", err)
	}
	b, err := NewCrypto()
	if err != nil {
		t.Fatalf("generating keypair: %v", err)
	}
	if a.PublicKeyHex() == b.PublicKeyHex() {
		t.Error("two generated keypairs share a public key")
	}
}

func TestCrypto_EncryptDecryptRoundTrip(t *testing.T) {
	c, err := NewCrypto()
	if err != nil {
		t.Fatalf("generating keypair: %v", err)
	}

	plaintext := []byte(`{"triggerPrice":"20000"}`)
	ciphertext, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypting: %v", err)
	}
	if strings.Contains(string(ciphertext), "20000") {
		t.Error("ciphertext leaks plaintext")
	}

	got, err := c.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decrypting: %v", err)
	}
	if string(got) != string(plaintext) {
		t.Errorf("round trip mismatch: got %q", got)
	}
}

func TestCrypto_DecryptGarbage(t *testing.T) {
	c, err := NewCrypto()
	if err != nil {
		t.Fatalf("generating keypair: %v", err)
	}

	for name, ct := range map[string][]byte{
		"empty":        {},
		"short":        {0x04, 0x01, 0x02},
		"not a point":  make([]byte, 200),
		"random bytes": []byte(strings.Repeat("garbage-bytes!", 12)),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := c.Decrypt(ct); err == nil {
				t.Error("expected decryption error, got nil")
			}
		})
	}
}

func TestCrypto_WrongKeyFails(t *testing.T) {
	a, _ := NewCrypto()
	b, _ := NewCrypto()

	ct, err := a.Encrypt([]byte(`{"triggerPrice":"20000"}`))
	if err != nil {
		t.Fatalf("encrypting: %v", err)
	}
	if _, err := b.Decrypt(ct); err == nil {
		t.Error("expected decryption with wrong key to fail")
	}
}

// eciesVector is the committed test fixture: a keypair plus a ciphertext
// produced by this package's own Encrypt (go-ethereum crypto/ecies,
// secp256k1 → AES-128-CTR + HMAC-SHA-256, NIST concat KDF).
type eciesVector struct {
	PrivateKeyHex string `json:"privateKeyHex"`
	PublicKeyHex  string `json:"publicKeyHex"`
	Plaintext     string `json:"plaintext"`
	CiphertextHex string `json:"ciphertextHex"`
}

const fixturePath = "testdata/ecies_vector.json"

// TestGenerateFixture regenerates the committed vector. Run explicitly:
//
//	GEN_ECIES_FIXTURE=1 go test ./internal/extension/ -run TestGenerateFixture
func TestGenerateFixture(t *testing.T) {
	if os.Getenv("GEN_ECIES_FIXTURE") == "" {
		t.Skip("set GEN_ECIES_FIXTURE=1 to regenerate the fixture")
	}
	c, err := NewCrypto()
	if err != nil {
		t.Fatalf("generating keypair: %v", err)
	}
	plaintext := `{"triggerPrice":"20000"}`
	ct, err := c.Encrypt([]byte(plaintext))
	if err != nil {
		t.Fatalf("encrypting: %v", err)
	}
	v := eciesVector{
		PrivateKeyHex: c.PrivateKeyHex(),
		PublicKeyHex:  c.PublicKeyHex(),
		Plaintext:     plaintext,
		CiphertextHex: "0x" + hex.EncodeToString(ct),
	}
	b, _ := json.MarshalIndent(v, "", "  ")
	if err := os.MkdirAll(filepath.Dir(fixturePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixturePath, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s", fixturePath)
}

func loadFixture(t *testing.T) eciesVector {
	t.Helper()
	b, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("reading fixture (regenerate with GEN_ECIES_FIXTURE=1): %v", err)
	}
	var v eciesVector
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}
	return v
}

func TestCrypto_FixtureVector(t *testing.T) {
	v := loadFixture(t)

	c, err := NewCryptoFromHex(v.PrivateKeyHex)
	if err != nil {
		t.Fatalf("importing fixture key: %v", err)
	}
	if c.PublicKeyHex() != v.PublicKeyHex {
		t.Errorf("fixture pubkey mismatch: got %s want %s", c.PublicKeyHex(), v.PublicKeyHex)
	}

	ct, err := hex.DecodeString(strings.TrimPrefix(v.CiphertextHex, "0x"))
	if err != nil {
		t.Fatalf("fixture ciphertext hex: %v", err)
	}
	got, err := c.Decrypt(ct)
	if err != nil {
		t.Fatalf("decrypting fixture: %v", err)
	}
	if string(got) != v.Plaintext {
		t.Errorf("fixture plaintext mismatch: got %q want %q", got, v.Plaintext)
	}

	// The fixture plaintext must parse as a valid trigger payload.
	trigger, err := ParseTriggerPlaintext(got)
	if err != nil {
		t.Fatalf("parsing fixture plaintext: %v", err)
	}
	if trigger.Cmp(big.NewInt(20000)) != 0 {
		t.Errorf("expected trigger 20000, got %s", trigger)
	}
}

func TestParseTriggerPlaintext(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    *big.Int
		wantErr bool
	}{
		{name: "valid", in: `{"triggerPrice":"20000"}`, want: big.NewInt(20000)},
		{name: "valid large", in: `{"triggerPrice":"123456789012345678901234567890"}`, want: bigFromString(t, "123456789012345678901234567890")},
		{name: "zero", in: `{"triggerPrice":"0"}`, wantErr: true},
		{name: "negative", in: `{"triggerPrice":"-5"}`, wantErr: true},
		{name: "decimal point", in: `{"triggerPrice":"0.02"}`, wantErr: true},
		{name: "not a number", in: `{"triggerPrice":"abc"}`, wantErr: true},
		{name: "empty string", in: `{"triggerPrice":""}`, wantErr: true},
		{name: "missing field", in: `{}`, wantErr: true},
		{name: "unknown field", in: `{"triggerPrice":"20000","owner":"0x1"}`, wantErr: true},
		{name: "bad json", in: `{trigger`, wantErr: true},
		{name: "number not string", in: `{"triggerPrice":20000}`, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseTriggerPlaintext([]byte(tc.in))
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got %s", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Cmp(tc.want) != 0 {
				t.Errorf("got %s want %s", got, tc.want)
			}
		})
	}
}

func TestParseOrderPlaintext_TrailingStop(t *testing.T) {
	policy, err := ParseOrderPlaintext([]byte(`{"strategy":"trailing","trailBps":500}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.TrailBps != 500 || policy.TriggerPrice != nil {
		t.Fatalf("unexpected policy: %+v", policy)
	}
	for _, payload := range []string{
		`{"strategy":"trailing","trailBps":25}`,
		`{"strategy":"trailing","trailBps":5000}`,
	} {
		if _, err := ParseOrderPlaintext([]byte(payload)); err != nil {
			t.Fatalf("expected boundary trail to pass: %s: %v", payload, err)
		}
	}
	for _, payload := range []string{
		`{"trailBps":500}`,
		`{"strategy":"trailing","trailBps":1}`,
		`{"strategy":"trailing","trailBps":5001}`,
	} {
		if _, err := ParseOrderPlaintext([]byte(payload)); err == nil {
			t.Fatalf("expected invalid trail to fail: %s", payload)
		}
	}
}

func TestParseOrderPlaintext_RejectsAmbiguousOrUnsafePayloads(t *testing.T) {
	overflow := new(big.Int).Lsh(big.NewInt(1), 256).String()
	cases := []string{
		`{"strategy":"fixed","triggerPrice":"20000","trailBps":500}`,
		`{"strategy":"trailing","triggerPrice":"20000","trailBps":500}`,
		`{"strategy":"fixed","triggerPrice":"20000"}{"strategy":"fixed","triggerPrice":"1"}`,
		`{"strategy":"fixed","triggerPrice":"20000","triggerPrice":"1"}`,
		`{"Strategy":"fixed","triggerPrice":"20000"}`,
		`{"strategy":"fixed","triggerPrice":"` + overflow + `"}`,
	}
	for _, payload := range cases {
		if _, err := ParseOrderPlaintext([]byte(payload)); err == nil {
			t.Errorf("expected payload to fail: %s", payload)
		}
	}
}

func TestParseOrderPlaintext_TaggedFixedAndLegacyFixed(t *testing.T) {
	for _, payload := range []string{
		`{"strategy":"fixed","triggerPrice":"20000"}`,
		`{"triggerPrice":"20000"}`,
		`{"strategy":"fixed","triggerPrice":"20000"}        `,
	} {
		policy, err := ParseOrderPlaintext([]byte(payload))
		if err != nil {
			t.Fatalf("expected fixed payload to pass: %s: %v", payload, err)
		}
		if policy.TriggerPrice.Cmp(big.NewInt(20_000)) != 0 || policy.TrailBps != 0 {
			t.Fatalf("unexpected policy for %s: %+v", payload, policy)
		}
	}
}

func bigFromString(t *testing.T, s string) *big.Int {
	t.Helper()
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		t.Fatalf("bad big int literal %q", s)
	}
	return v
}
