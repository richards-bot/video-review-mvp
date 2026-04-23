/**
 * Tests for nonce/AAD construction and fixture decryption.
 * Nonce/AAD functions mirror decrypt-worker.js (§3.3, §3.4 of SPEC.md).
 * Fixture decryption verifies Go-encrypted bytes round-trip in JS Web Crypto.
 */

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const FIXTURE_DIR = join(__dirname, '..', 'fixtures');

// --- Functions under test (mirrored from decrypt-worker.js) ---

function chunkNonce(chunkIndex) {
  const nonce = new Uint8Array(12);
  // bytes 0-3 are 0x00 (chunk prefix)
  const view = new DataView(nonce.buffer);
  view.setBigUint64(4, BigInt(chunkIndex), false); // big-endian
  return nonce;
}

function manifestNonce() {
  const nonce = new Uint8Array(12);
  nonce[0] = 0xFF;
  nonce[1] = 0xFF;
  nonce[2] = 0xFF;
  nonce[3] = 0xFF;
  // bytes 4-11 are 0x00
  return nonce;
}

function chunkAAD(assetId, chunkIndex) {
  return new TextEncoder().encode(`${assetId}|${chunkIndex}`);
}

function manifestAAD(assetId) {
  return new TextEncoder().encode(`${assetId}|manifest`);
}

// --- nonce construction ---

describe('chunkNonce', () => {
  it('produces 12-byte nonce', () => {
    expect(chunkNonce(0)).toHaveLength(12);
  });

  it('first 4 bytes are 0x00 for chunk 0', () => {
    const n = chunkNonce(0);
    expect(n[0]).toBe(0);
    expect(n[1]).toBe(0);
    expect(n[2]).toBe(0);
    expect(n[3]).toBe(0);
  });

  it('encodes chunk index as big-endian uint64 in bytes 4-11', () => {
    const n = chunkNonce(1);
    // chunk index 1 → last byte of the uint64 is 1, rest 0
    expect(n[4]).toBe(0);
    expect(n[11]).toBe(1);
  });

  it('encodes large chunk index correctly', () => {
    // chunk index 256 → bytes 4-10 are 0, byte 10 is 1, byte 11 is 0
    const n = chunkNonce(256);
    expect(n[10]).toBe(1);
    expect(n[11]).toBe(0);
  });

  it('chunk 0 and chunk 1 produce different nonces', () => {
    const n0 = chunkNonce(0);
    const n1 = chunkNonce(1);
    expect(Array.from(n0)).not.toEqual(Array.from(n1));
  });

  it('produces deterministic output for same index', () => {
    const a = chunkNonce(42);
    const b = chunkNonce(42);
    expect(Array.from(a)).toEqual(Array.from(b));
  });
});

describe('manifestNonce', () => {
  it('produces 12-byte nonce', () => {
    expect(manifestNonce()).toHaveLength(12);
  });

  it('first 4 bytes are 0xFF (distinct from chunk prefix)', () => {
    const n = manifestNonce();
    expect(n[0]).toBe(0xFF);
    expect(n[1]).toBe(0xFF);
    expect(n[2]).toBe(0xFF);
    expect(n[3]).toBe(0xFF);
  });

  it('bytes 4-11 are 0x00', () => {
    const n = manifestNonce();
    for (let i = 4; i < 12; i++) {
      expect(n[i]).toBe(0);
    }
  });

  it('differs from chunk 0 nonce', () => {
    expect(Array.from(manifestNonce())).not.toEqual(Array.from(chunkNonce(0)));
  });
});

// --- AAD construction ---

describe('chunkAAD', () => {
  it('encodes as assetId|chunkIndex', () => {
    const aad = chunkAAD('aabbccddeeff00112233445566778899', 0);
    const text = new TextDecoder().decode(aad);
    expect(text).toBe('aabbccddeeff00112233445566778899|0');
  });

  it('uses the numeric chunk index (not padded)', () => {
    const aad = chunkAAD('aabbccddeeff00112233445566778899', 42);
    const text = new TextDecoder().decode(aad);
    expect(text).toBe('aabbccddeeff00112233445566778899|42');
  });

  it('returns a Uint8Array', () => {
    expect(chunkAAD('x', 0)).toBeInstanceOf(Uint8Array);
  });
});

describe('manifestAAD', () => {
  it('encodes as assetId|manifest', () => {
    const aad = manifestAAD('aabbccddeeff00112233445566778899');
    const text = new TextDecoder().decode(aad);
    expect(text).toBe('aabbccddeeff00112233445566778899|manifest');
  });

  it('differs from chunk 0 AAD for the same asset', () => {
    const id = 'aabbccddeeff00112233445566778899';
    expect(Array.from(manifestAAD(id))).not.toEqual(Array.from(chunkAAD(id, 0)));
  });

  it('returns a Uint8Array', () => {
    expect(manifestAAD('x')).toBeInstanceOf(Uint8Array);
  });
});

// --- Fixture decryption (Go-encrypt ↔ JS-decrypt interop) ---

describe('fixture decryption', () => {
  const ASSET_ID = 'aabbccddeeff00112233445566778899';
  const CEK_HEX = '0000000000000000000000000000000000000000000000000000000000000000';

  function hexToBytes(hex) {
    const bytes = new Uint8Array(hex.length / 2);
    for (let i = 0; i < hex.length; i += 2) {
      bytes[i / 2] = parseInt(hex.slice(i, i + 2), 16);
    }
    return bytes;
  }

  async function importKey(cekBytes) {
    return crypto.subtle.importKey(
      'raw',
      cekBytes,
      { name: 'AES-GCM' },
      false,
      ['decrypt']
    );
  }

  async function decryptChunk(key, assetId, chunkIndex, ciphertext) {
    return crypto.subtle.decrypt(
      {
        name: 'AES-GCM',
        iv: chunkNonce(chunkIndex),
        additionalData: chunkAAD(assetId, chunkIndex),
      },
      key,
      ciphertext
    );
  }

  async function decryptManifest(key, assetId, ciphertext) {
    return crypto.subtle.decrypt(
      {
        name: 'AES-GCM',
        iv: manifestNonce(),
        additionalData: manifestAAD(assetId),
      },
      key,
      ciphertext
    );
  }

  it('decrypts fixture chunk 0 to expected plaintext', async () => {
    const cekBytes = hexToBytes(CEK_HEX);
    const key = await importKey(cekBytes);
    const ciphertext = readFileSync(join(FIXTURE_DIR, 'fixture-chunk-000000.enc'));
    const plaintext = await decryptChunk(key, ASSET_ID, 0, ciphertext);
    expect(new TextDecoder().decode(plaintext)).toBe('chunk0data');
  });

  it('decrypts fixture chunk 1 to expected plaintext', async () => {
    const cekBytes = hexToBytes(CEK_HEX);
    const key = await importKey(cekBytes);
    const ciphertext = readFileSync(join(FIXTURE_DIR, 'fixture-chunk-000001.enc'));
    const plaintext = await decryptChunk(key, ASSET_ID, 1, ciphertext);
    expect(new TextDecoder().decode(plaintext)).toBe('chunk1data');
  });

  it('decrypts fixture chunk 2 to expected plaintext', async () => {
    const cekBytes = hexToBytes(CEK_HEX);
    const key = await importKey(cekBytes);
    const ciphertext = readFileSync(join(FIXTURE_DIR, 'fixture-chunk-000002.enc'));
    const plaintext = await decryptChunk(key, ASSET_ID, 2, ciphertext);
    expect(new TextDecoder().decode(plaintext)).toBe('chunk2data');
  });

  it('decrypts fixture manifest and parses expected fields', async () => {
    const cekBytes = hexToBytes(CEK_HEX);
    const key = await importKey(cekBytes);
    const ciphertext = readFileSync(join(FIXTURE_DIR, 'fixture-manifest.enc'));
    const plaintext = await decryptManifest(key, ASSET_ID, ciphertext);
    const manifest = JSON.parse(new TextDecoder().decode(plaintext));

    expect(manifest.version).toBe(1);
    expect(manifest.asset_id).toBe(ASSET_ID);
    expect(manifest.chunk_count).toBe(3);
    expect(manifest.source_filename).toBe('fixture.mp4');
    expect(manifest.codec.mime).toContain('video/mp4');
  });

  it('rejects chunk 0 ciphertext when decrypted as chunk 1 (AAD binding)', async () => {
    const cekBytes = hexToBytes(CEK_HEX);
    const key = await importKey(cekBytes);
    const ciphertext = readFileSync(join(FIXTURE_DIR, 'fixture-chunk-000000.enc'));
    // Attempt to decrypt chunk 0's ciphertext using chunk index 1 — AAD mismatch
    await expect(decryptChunk(key, ASSET_ID, 1, ciphertext)).rejects.toThrow();
  });

  it('rejects chunk ciphertext with wrong asset ID (AAD binding)', async () => {
    const cekBytes = hexToBytes(CEK_HEX);
    const key = await importKey(cekBytes);
    const ciphertext = readFileSync(join(FIXTURE_DIR, 'fixture-chunk-000000.enc'));
    await expect(decryptChunk(key, 'ffffffffffffffffffffffffffffffff', 0, ciphertext)).rejects.toThrow();
  });

  it('rejects manifest ciphertext decrypted with wrong CEK', async () => {
    const wrongCEK = new Uint8Array(32).fill(0xFF);
    const key = await importKey(wrongCEK);
    const ciphertext = readFileSync(join(FIXTURE_DIR, 'fixture-manifest.enc'));
    await expect(decryptManifest(key, ASSET_ID, ciphertext)).rejects.toThrow();
  });

  it('verifies fixture.json metadata matches decrypted manifest', async () => {
    const fixtureSpec = JSON.parse(readFileSync(join(FIXTURE_DIR, 'fixture.json'), 'utf8'));
    const cekBytes = hexToBytes(CEK_HEX);
    const key = await importKey(cekBytes);
    const ciphertext = readFileSync(join(FIXTURE_DIR, 'fixture-manifest.enc'));
    const plaintext = await decryptManifest(key, ASSET_ID, ciphertext);
    const manifest = JSON.parse(new TextDecoder().decode(plaintext));

    expect(manifest.asset_id).toBe(fixtureSpec.asset_id);
    expect(manifest.chunk_count).toBe(fixtureSpec.chunk_count);
    // fixture.json records expected plaintext per chunk
    expect(fixtureSpec.chunk_plaintexts).toHaveLength(3);
    expect(fixtureSpec.chunk_plaintexts[0]).toBe('chunk0data');
  });
});
