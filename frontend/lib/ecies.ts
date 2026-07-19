// ECIES encryptor wire-compatible with go-ethereum's crypto/ecies
// (ECIES_AES128_SHA256 for secp256k1). eciesjs is NOT compatible — it uses
// HKDF + AES-GCM. This implements the exact go-ethereum construction:
//
//   ciphertext = 0x04 ‖ ephemeralPubXY (64) ‖ IV (16) ‖ AES-128-CTR ct ‖ HMAC-SHA-256 tag (32)
//
//   z  = X coordinate of (ephemeralPriv * recipientPub), 32-byte big-endian
//   K  = NIST SP 800-56 concatKDF(SHA-256, z, s1 = nil, kdLen = 32)
//   Ke = K[0:16]                    (AES-128-CTR key)
//   Km = SHA-256(K[16:32])          (HMAC key — go-ethereum hashes it)
//   tag = HMAC-SHA-256(Km, IV ‖ ct) (s2 = nil)
//
// Conformance target: internal/extension/testdata/ecies_vector.json
// (round-trip verified in lib/ecies.test.ts against a Go-produced ciphertext).
import { secp256k1 } from "@noble/curves/secp256k1.js";
import { sha256 } from "@noble/hashes/sha2.js";
import { hmac } from "@noble/hashes/hmac.js";
import { ctr } from "@noble/ciphers/aes.js";
import {
  bytesToHex,
  concatBytes,
  hexToBytes,
  randomBytes,
} from "@noble/hashes/utils.js";

// Equal-length plaintext prevents ciphertext length from revealing whether a
// placement contains a fixed or trailing policy. JSON is padded with spaces,
// which Go's strict decoder accepts only after the single top-level object.
export const POLICY_PLAINTEXT_BYTES = 128;

function encodePaddedPolicy(policy: object): Uint8Array {
  const json = new TextEncoder().encode(JSON.stringify(policy));
  if (json.length > POLICY_PLAINTEXT_BYTES) {
    throw new Error("encrypted policy exceeds fixed plaintext envelope");
  }
  const padded = new Uint8Array(POLICY_PLAINTEXT_BYTES).fill(0x20);
  padded.set(json);
  return padded;
}

/** NIST SP 800-56 concatenation KDF as implemented by go-ethereum:
 *  K = SHA-256(counter_1 ‖ z ‖ s1) ‖ SHA-256(counter_2 ‖ z ‖ s1) ‖ … */
export function concatKDF(z: Uint8Array, kdLen: number): Uint8Array {
  const rounds: Uint8Array[] = [];
  for (let counter = 1; rounds.length * 32 < kdLen; counter++) {
    const counterBE = new Uint8Array([
      counter >>> 24,
      (counter >>> 16) & 0xff,
      (counter >>> 8) & 0xff,
      counter & 0xff,
    ]);
    rounds.push(sha256(concatBytes(counterBE, z)));
  }
  return concatBytes(...rounds).slice(0, kdLen);
}

/** Derive (Ke, Km) from an ECDH shared X coordinate, go-ethereum style. */
export function deriveKeys(z: Uint8Array): { ke: Uint8Array; km: Uint8Array } {
  const k = concatKDF(z, 32);
  return { ke: k.slice(0, 16), km: sha256(k.slice(16, 32)) };
}

/** Encrypt `plaintext` to an uncompressed secp256k1 public key (65 bytes,
 *  0x04-prefixed). `ephemeralPriv` / `iv` are injectable for tests only. */
export function eciesEncrypt(
  recipientPub: Uint8Array,
  plaintext: Uint8Array,
  ephemeralPriv?: Uint8Array,
  iv?: Uint8Array,
): Uint8Array {
  const eph = ephemeralPriv ?? secp256k1.utils.randomSecretKey();
  const ephPub = secp256k1.getPublicKey(eph, false); // 65B: 0x04 ‖ X ‖ Y
  const shared = secp256k1.getSharedSecret(eph, recipientPub, false);
  const { ke, km } = deriveKeys(shared.slice(1, 33)); // z = X, 32B BE
  const ivBytes = iv ?? randomBytes(16);
  const ct = ctr(ke, ivBytes).encrypt(plaintext);
  const em = concatBytes(ivBytes, ct);
  return concatBytes(ephPub, em, hmac(sha256, km, em));
}

/** Encrypt the DarkStop trigger payload to the TEE's pubkey (0x04… hex).
 *  `triggerPrice6` is USD/FLR as an integer string scaled to 6 decimals.
 *  Returns 0x-prefixed ciphertext hex ready for placeOrder(bytes). */
export function encryptTriggerPrice(
  teePubKeyHex: string,
  triggerPrice6: string,
): `0x${string}` {
  if (!/^[1-9]\d*$/.test(triggerPrice6)) {
    throw new Error(`trigger price must be a positive integer string, got "${triggerPrice6}"`);
  }
  if (BigInt(triggerPrice6) >= (1n << 256n)) {
    throw new Error("trigger price exceeds uint256");
  }
  const pub = hexToBytes(teePubKeyHex.replace(/^0x/, ""));
  if (pub.length !== 65 || pub[0] !== 0x04) {
    throw new Error("TEE public key must be 65-byte uncompressed (0x04-prefixed)");
  }
  const plaintext = encodePaddedPolicy({ strategy: "fixed", triggerPrice: triggerPrice6 });
  return `0x${bytesToHex(eciesEncrypt(pub, plaintext))}`;
}

export function encryptTrailingStop(
  teePubKeyHex: string,
  trailBps: number,
): `0x${string}` {
  if (!Number.isInteger(trailBps) || trailBps < 25 || trailBps > 5000) {
    throw new Error("trailing distance must be between 0.25% and 50%");
  }
  const pub = hexToBytes(teePubKeyHex.replace(/^0x/, ""));
  if (pub.length !== 65 || pub[0] !== 0x04) {
    throw new Error("TEE public key must be 65-byte uncompressed (0x04-prefixed)");
  }
  const plaintext = encodePaddedPolicy({ strategy: "trailing", trailBps });
  return `0x${bytesToHex(eciesEncrypt(pub, plaintext))}`;
}
