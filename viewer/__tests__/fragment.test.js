/**
 * Tests for URL fragment parsing, validation, and CEK decoding.
 * These functions mirror the implementations in viewer.js (§4.4, §6.2 of SPEC.md).
 */

import { describe, it, expect } from 'vitest';

// --- Functions under test (mirrored from viewer.js) ---

function parseFragment(hash) {
  if (!hash || hash.length < 2) return null;
  const raw = hash.startsWith('#') ? hash.slice(1) : hash;
  const params = {};
  for (const part of raw.split('&')) {
    const eq = part.indexOf('=');
    if (eq === -1) continue;
    params[part.slice(0, eq)] = part.slice(eq + 1);
  }
  return params;
}

function validateParams(params) {
  if (!params) return 'missing fragment';
  if (params.v !== '1') return 'unsupported URL version';
  if (!/^[0-9a-f]{32}$/.test(params.a)) return 'invalid asset_id';
  if (!params.k || params.k.length !== 43) return 'invalid CEK length';
  if (!params.b || params.b.length === 0) return 'missing bucket';
  if (!params.r || params.r.length === 0) return 'missing region';
  return null;
}

function decodeCEK(b64url) {
  const b64 = b64url.replace(/-/g, '+').replace(/_/g, '/');
  const padded = b64 + '='.repeat((4 - b64.length % 4) % 4);
  const bin = atob(padded);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return bytes;
}

// --- Test data ---

const VALID_ASSET_ID = 'aabbccddeeff00112233445566778899';
// all-zeros CEK, URL-safe base64 without padding (43 chars for 32 bytes)
const VALID_CEK_B64 = 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA';
const VALID_FRAGMENT =
  `#v=1&a=${VALID_ASSET_ID}&k=${VALID_CEK_B64}&b=my-bucket&r=eu-west-2`;

// --- parseFragment ---

describe('parseFragment', () => {
  it('parses a valid fragment', () => {
    const p = parseFragment(VALID_FRAGMENT);
    expect(p.v).toBe('1');
    expect(p.a).toBe(VALID_ASSET_ID);
    expect(p.k).toBe(VALID_CEK_B64);
    expect(p.b).toBe('my-bucket');
    expect(p.r).toBe('eu-west-2');
  });

  it('strips leading # before parsing', () => {
    const p = parseFragment('#v=1&a=x&b=y');
    expect(p.v).toBe('1');
    expect(p.a).toBe('x');
  });

  it('handles fragment without leading #', () => {
    const p = parseFragment('v=1&a=x');
    expect(p.v).toBe('1');
  });

  it('returns null for empty string', () => {
    expect(parseFragment('')).toBeNull();
  });

  it('returns null for null input', () => {
    expect(parseFragment(null)).toBeNull();
  });

  it('returns null for bare # with no params', () => {
    expect(parseFragment('#')).toBeNull();
  });

  it('skips parts without = sign', () => {
    const p = parseFragment('#v=1&malformed&a=x');
    expect(p.v).toBe('1');
    expect(p.a).toBe('x');
    expect(p.malformed).toBeUndefined();
  });

  it('handles value containing = sign', () => {
    const p = parseFragment('#v=1&k=base64=padded');
    expect(p.k).toBe('base64=padded');
  });
});

// --- validateParams ---

describe('validateParams', () => {
  function validParams() {
    return { v: '1', a: VALID_ASSET_ID, k: VALID_CEK_B64, b: 'bucket', r: 'eu-west-2' };
  }

  it('returns null for valid params', () => {
    expect(validateParams(validParams())).toBeNull();
  });

  it('returns error for null params', () => {
    expect(validateParams(null)).toBe('missing fragment');
  });

  it('rejects wrong version', () => {
    expect(validateParams({ ...validParams(), v: '2' })).toBe('unsupported URL version');
  });

  it('rejects missing version', () => {
    const p = validParams();
    delete p.v;
    expect(validateParams(p)).toBe('unsupported URL version');
  });

  it('rejects asset_id that is too short', () => {
    expect(validateParams({ ...validParams(), a: 'abc' })).toBe('invalid asset_id');
  });

  it('rejects asset_id with uppercase hex', () => {
    expect(validateParams({ ...validParams(), a: 'AABBCCDDEEFF00112233445566778899' })).toBe('invalid asset_id');
  });

  it('rejects asset_id with non-hex chars', () => {
    expect(validateParams({ ...validParams(), a: 'zzbccddeeff00112233445566778899' })).toBe('invalid asset_id');
  });

  it('rejects CEK that is too short (42 chars)', () => {
    expect(validateParams({ ...validParams(), k: 'A'.repeat(42) })).toBe('invalid CEK length');
  });

  it('rejects CEK that is too long (44 chars)', () => {
    expect(validateParams({ ...validParams(), k: 'A'.repeat(44) })).toBe('invalid CEK length');
  });

  it('rejects missing bucket', () => {
    expect(validateParams({ ...validParams(), b: '' })).toBe('missing bucket');
  });

  it('rejects missing region', () => {
    expect(validateParams({ ...validParams(), r: '' })).toBe('missing region');
  });
});

// --- decodeCEK ---

describe('decodeCEK', () => {
  it('decodes all-zeros CEK to 32 zero bytes', () => {
    const bytes = decodeCEK(VALID_CEK_B64);
    expect(bytes.length).toBe(32);
    expect(Array.from(bytes).every(b => b === 0)).toBe(true);
  });

  it('decodes URL-safe base64 (- replaced with +)', () => {
    // encode [0xFB, 0xFF] = base64 "+/8" (standard) → "-_8" (URL-safe)
    // Use a known mapping: 0xFB = 251, 0xFF = 255 → base64 is "+/8" → url-safe "-_8"
    // Pad to 32 bytes for a realistic test
    const allFF = new Uint8Array(32).fill(0xFF);
    // btoa encodes to standard base64 with + and /
    const b64standard = btoa(String.fromCharCode(...allFF));
    const b64url = b64standard.replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
    const decoded = decodeCEK(b64url);
    expect(decoded.length).toBe(32);
    expect(Array.from(decoded).every(b => b === 0xFF)).toBe(true);
  });

  it('returns a Uint8Array', () => {
    const bytes = decodeCEK(VALID_CEK_B64);
    expect(bytes).toBeInstanceOf(Uint8Array);
  });

  it('round-trips through encode/decode', () => {
    const original = new Uint8Array(32);
    for (let i = 0; i < 32; i++) original[i] = i * 8;
    const b64standard = btoa(String.fromCharCode(...original));
    const b64url = b64standard.replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
    const decoded = decodeCEK(b64url);
    expect(Array.from(decoded)).toEqual(Array.from(original));
  });
});
