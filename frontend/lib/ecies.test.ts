// Conformance tests for the go-ethereum-compatible ECIES encryptor.
//
// Strategy: the fixture internal/extension/testdata/ecies_vector.json holds a
// ciphertext produced by go-ethereum's crypto/ecies (the exact code the TEE
// extension decrypts with). ECIES uses a random ephemeral key + random IV, so
// the fixture ciphertext bytes cannot be reproduced. Instead:
//
//   1. a minimal decryptor (test-only, below) decrypts + MAC-verifies the
//      Go-produced fixture ciphertext — proving KDF, key-split, Km hashing,
//      HMAC coverage and wire layout all match go-ethereum, and
//   2. our encryptor's output round-trips through that same validated
//      decryptor — proving encrypt emits the identical construction.
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import { secp256k1 } from "@noble/curves/secp256k1.js";
import { sha256 } from "@noble/hashes/sha2.js";
import { hmac } from "@noble/hashes/hmac.js";
import { ctr } from "@noble/ciphers/aes.js";
import { hexToBytes, bytesToHex } from "@noble/hashes/utils.js";
import { concatKDF, deriveKeys, eciesEncrypt, encryptTriggerPrice } from "./ecies";

const fixture: {
  privateKeyHex: string;
  publicKeyHex: string;
  plaintext: string;
  ciphertextHex: string;
} = JSON.parse(
  readFileSync(
    join(__dirname, "../../internal/extension/testdata/ecies_vector.json"),
    "utf8",
  ),
);

const priv = hexToBytes(fixture.privateKeyHex.slice(2));
const pub = hexToBytes(fixture.publicKeyHex.slice(2));

/** Minimal go-ethereum ECIES decryptor — test-only, used to validate the
 *  encryptor. Throws on MAC mismatch, exactly like the Go side. */
function eciesDecrypt(privateKey: Uint8Array, ciphertext: Uint8Array): Uint8Array {
  if (ciphertext.length < 65 + 16 + 32 + 1 || ciphertext[0] !== 0x04) {
    throw new Error("malformed ciphertext");
  }
  const ephPub = ciphertext.slice(0, 65);
  const em = ciphertext.slice(65, ciphertext.length - 32); // IV ‖ ct
  const tag = ciphertext.slice(ciphertext.length - 32);
  const shared = secp256k1.getSharedSecret(privateKey, ephPub, false);
  const { ke, km } = deriveKeys(shared.slice(1, 33));
  const expected = hmac(sha256, km, em);
  if (bytesToHex(expected) !== bytesToHex(tag)) throw new Error("invalid MAC");
  return ctr(ke, em.slice(0, 16)).decrypt(em.slice(16));
}

describe("fixture sanity", () => {
  it("public key in fixture matches its private key", () => {
    expect(bytesToHex(secp256k1.getPublicKey(priv, false))).toBe(
      bytesToHex(pub),
    );
  });
});

describe("wire format structure", () => {
  const ciphertext = eciesEncrypt(pub, new TextEncoder().encode(fixture.plaintext));

  it("is 0x04 ‖ ephPubXY(64) ‖ IV(16) ‖ ct ‖ tag(32)", () => {
    expect(ciphertext[0]).toBe(0x04);
    expect(ciphertext.length).toBe(65 + 16 + fixture.plaintext.length + 32);
  });

  it("matches the Go-produced fixture ciphertext's length and prefix", () => {
    const goCiphertext = hexToBytes(fixture.ciphertextHex.slice(2));
    expect(ciphertext.length).toBe(goCiphertext.length);
    expect(goCiphertext[0]).toBe(0x04);
  });

  it("embeds a valid secp256k1 point as the ephemeral pubkey", () => {
    expect(() =>
      secp256k1.Point.fromBytes(ciphertext.slice(0, 65)),
    ).not.toThrow();
  });
});

describe("conformance against Go-produced ciphertext", () => {
  it("decrypts the fixture ciphertext produced by go-ethereum crypto/ecies", () => {
    const plaintext = eciesDecrypt(priv, hexToBytes(fixture.ciphertextHex.slice(2)));
    expect(new TextDecoder().decode(plaintext)).toBe(fixture.plaintext);
  });

  it("rejects the fixture ciphertext when the tag is corrupted", () => {
    const corrupted = hexToBytes(fixture.ciphertextHex.slice(2));
    corrupted[corrupted.length - 1] ^= 0xff;
    expect(() => eciesDecrypt(priv, corrupted)).toThrow(/invalid MAC/);
  });
});

describe("encrypt round-trip through the Go-validated decryptor", () => {
  it("round-trips the fixture plaintext", () => {
    const ct = eciesEncrypt(pub, new TextEncoder().encode(fixture.plaintext));
    expect(new TextDecoder().decode(eciesDecrypt(priv, ct))).toBe(
      fixture.plaintext,
    );
  });

  it("round-trips with a fixed ephemeral key and IV (deterministic)", () => {
    const eph = sha256(new TextEncoder().encode("darkstop-test-ephemeral"));
    const iv = new Uint8Array(16).fill(7);
    const a = eciesEncrypt(pub, new TextEncoder().encode("hi"), eph, iv);
    const b = eciesEncrypt(pub, new TextEncoder().encode("hi"), eph, iv);
    expect(bytesToHex(a)).toBe(bytesToHex(b));
    expect(new TextDecoder().decode(eciesDecrypt(priv, a))).toBe("hi");
  });
});

describe("concatKDF", () => {
  it("is single-round SHA-256(0x00000001 ‖ z) for kdLen 32", () => {
    const z = new Uint8Array(32).fill(0xab);
    const expected = sha256(
      new Uint8Array([0, 0, 0, 1, ...z]),
    );
    expect(bytesToHex(concatKDF(z, 32))).toBe(bytesToHex(expected));
  });
});

describe("encryptTriggerPrice", () => {
  it("produces placeOrder-ready hex the TEE key can decrypt to canonical JSON", () => {
    const hex = encryptTriggerPrice(fixture.publicKeyHex, "20000");
    expect(hex).toMatch(/^0x04[0-9a-f]+$/);
    const plaintext = new TextDecoder().decode(
      eciesDecrypt(priv, hexToBytes(hex.slice(2))),
    );
    expect(plaintext).toBe('{"triggerPrice":"20000"}');
    expect(plaintext).toBe(fixture.plaintext);
  });

  it("rejects non-integer or non-positive trigger prices", () => {
    for (const bad of ["0", "-5", "1.5", "abc", "", "007"]) {
      expect(() => encryptTriggerPrice(fixture.publicKeyHex, bad)).toThrow();
    }
  });
});
