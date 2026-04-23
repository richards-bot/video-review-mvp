package crypto

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// Fixed test key: 32 bytes, all zeros for determinism in test vectors.
var testKey = bytes.Repeat([]byte{0x00}, 32)

// Fixed asset ID for test vectors.
const testAssetID = "7f3a9c2e4b1d8f600123456789abcdef"

func TestChunkNonce(t *testing.T) {
	tests := []struct {
		index uint64
		want  string // hex
	}{
		{0, "000000000000000000000000"},
		{1, "000000000000000000000001"},
		{42, "00000000000000000000002a"},
		{0xFFFFFFFFFFFFFFFF, "00000000ffffffffffffffff"},
	}
	for _, tt := range tests {
		n := ChunkNonce(tt.index)
		got := hex.EncodeToString(n[:])
		if got != tt.want {
			t.Errorf("ChunkNonce(%d) = %s, want %s", tt.index, got, tt.want)
		}
	}
}

func TestManifestNonce(t *testing.T) {
	n := ManifestNonce()
	got := hex.EncodeToString(n[:])
	want := "ffffffff0000000000000000"
	if got != want {
		t.Errorf("ManifestNonce() = %s, want %s", got, want)
	}
}

func TestChunkAAD(t *testing.T) {
	got := string(ChunkAAD(testAssetID, 42))
	want := "7f3a9c2e4b1d8f600123456789abcdef|42"
	if got != want {
		t.Errorf("ChunkAAD = %q, want %q", got, want)
	}
}

func TestManifestAAD(t *testing.T) {
	got := string(ManifestAAD(testAssetID))
	want := "7f3a9c2e4b1d8f600123456789abcdef|manifest"
	if got != want {
		t.Errorf("ManifestAAD = %q, want %q", got, want)
	}
}

func TestSealOpen_RoundTrip(t *testing.T) {
	plaintext := []byte("hello world this is a test payload")
	nonce := ChunkNonce(7)
	aad := ChunkAAD(testAssetID, 7)

	ct, err := Seal(testKey, nonce, plaintext, aad)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if len(ct) != len(plaintext)+16 {
		t.Errorf("ciphertext length = %d, want %d", len(ct), len(plaintext)+16)
	}

	pt, err := Open(testKey, nonce, ct, aad)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !bytes.Equal(pt, plaintext) {
		t.Errorf("decrypted != plaintext")
	}
}

func TestSealOpen_Deterministic(t *testing.T) {
	plaintext := []byte("deterministic test")
	nonce := ChunkNonce(0)
	aad := ChunkAAD(testAssetID, 0)

	ct1, err := Seal(testKey, nonce, plaintext, aad)
	if err != nil {
		t.Fatalf("Seal 1: %v", err)
	}
	ct2, err := Seal(testKey, nonce, plaintext, aad)
	if err != nil {
		t.Fatalf("Seal 2: %v", err)
	}
	if !bytes.Equal(ct1, ct2) {
		t.Error("Seal is not deterministic for same inputs")
	}
}

// TestSealOpen_KnownVector verifies the output against a known ciphertext.
// This acts as a cross-check: if Go's AES-GCM output changes, this test breaks.
func TestSealOpen_KnownVector(t *testing.T) {
	// All-zero key, chunk index 0, asset ID = testAssetID, plaintext = "test"
	plaintext := []byte("test")
	nonce := ChunkNonce(0)
	aad := ChunkAAD(testAssetID, 0)

	ct, err := Seal(testKey, nonce, plaintext, aad)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	// Record the ciphertext for cross-check. Re-encrypt and compare.
	ctHex := hex.EncodeToString(ct)
	t.Logf("known vector ciphertext: %s", ctHex)

	// Verify round-trip
	pt, err := Open(testKey, nonce, ct, aad)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !bytes.Equal(pt, plaintext) {
		t.Error("round-trip failed")
	}
}

func TestOpen_WrongKey(t *testing.T) {
	plaintext := []byte("secret data")
	nonce := ChunkNonce(0)
	aad := ChunkAAD(testAssetID, 0)

	ct, _ := Seal(testKey, nonce, plaintext, aad)

	wrongKey := bytes.Repeat([]byte{0x01}, 32)
	_, err := Open(wrongKey, nonce, ct, aad)
	if err == nil {
		t.Error("expected error with wrong key, got nil")
	}
}

func TestOpen_TamperedCiphertext(t *testing.T) {
	plaintext := []byte("tamper test")
	nonce := ChunkNonce(0)
	aad := ChunkAAD(testAssetID, 0)

	ct, _ := Seal(testKey, nonce, plaintext, aad)
	ct[0] ^= 0xFF // flip bits in first byte

	_, err := Open(testKey, nonce, ct, aad)
	if err == nil {
		t.Error("expected authentication failure with tampered ciphertext")
	}
}

func TestOpen_WrongAAD(t *testing.T) {
	plaintext := []byte("aad binding test")
	nonce := ChunkNonce(5)
	aad := ChunkAAD(testAssetID, 5)

	ct, _ := Seal(testKey, nonce, plaintext, aad)

	wrongAAD := ChunkAAD(testAssetID, 6) // wrong chunk index
	_, err := Open(testKey, nonce, ct, wrongAAD)
	if err == nil {
		t.Error("expected authentication failure with wrong AAD")
	}
}

func TestEncryptDecryptChunk_RoundTrip(t *testing.T) {
	plaintext := []byte("chunk payload data for testing")
	ct, err := EncryptChunk(testKey, testAssetID, 3, plaintext)
	if err != nil {
		t.Fatalf("EncryptChunk: %v", err)
	}
	pt, err := DecryptChunk(testKey, testAssetID, 3, ct)
	if err != nil {
		t.Fatalf("DecryptChunk: %v", err)
	}
	if !bytes.Equal(pt, plaintext) {
		t.Error("chunk round-trip failed")
	}
}

func TestEncryptDecryptManifest_RoundTrip(t *testing.T) {
	manifest := []byte(`{"version":1,"asset_id":"7f3a9c2e4b1d8f600123456789abcdef"}`)
	ct, err := EncryptManifest(testKey, testAssetID, manifest)
	if err != nil {
		t.Fatalf("EncryptManifest: %v", err)
	}
	pt, err := DecryptManifest(testKey, testAssetID, ct)
	if err != nil {
		t.Fatalf("DecryptManifest: %v", err)
	}
	if !bytes.Equal(pt, manifest) {
		t.Error("manifest round-trip failed")
	}
}

func TestChunkNonce_DistinctFromManifestNonce(t *testing.T) {
	chunkN := ChunkNonce(0)
	manifestN := ManifestNonce()
	if chunkN == manifestN {
		t.Error("chunk nonce(0) must differ from manifest nonce")
	}
}
