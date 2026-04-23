package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
)

// ChunkNonce builds the 12-byte nonce for chunk encryption.
// Bytes 0-3: 0x00 0x00 0x00 0x00
// Bytes 4-11: big-endian uint64 chunk index
func ChunkNonce(chunkIndex uint64) [12]byte {
	var nonce [12]byte
	binary.BigEndian.PutUint64(nonce[4:], chunkIndex)
	return nonce
}

// ManifestNonce builds the 12-byte nonce for manifest encryption.
// Bytes 0-3: 0xFF 0xFF 0xFF 0xFF
// Bytes 4-11: 0x00
func ManifestNonce() [12]byte {
	var nonce [12]byte
	nonce[0] = 0xFF
	nonce[1] = 0xFF
	nonce[2] = 0xFF
	nonce[3] = 0xFF
	return nonce
}

// ChunkAAD builds the additional authenticated data for a chunk.
// Format: "<asset_id>|<chunk_index_decimal>"
func ChunkAAD(assetID string, chunkIndex uint64) []byte {
	return []byte(fmt.Sprintf("%s|%d", assetID, chunkIndex))
}

// ManifestAAD builds the additional authenticated data for the manifest.
// Format: "<asset_id>|manifest"
func ManifestAAD(assetID string) []byte {
	return []byte(assetID + "|manifest")
}

// Seal encrypts plaintext using AES-256-GCM with the provided key, nonce, and AAD.
// Returns ciphertext || 16-byte GCM tag.
func Seal(key []byte, nonce [12]byte, plaintext, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}
	return gcm.Seal(nil, nonce[:], plaintext, aad), nil
}

// Open decrypts ciphertext (ciphertext || tag) using AES-256-GCM.
func Open(key []byte, nonce [12]byte, ciphertext, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}
	plaintext, err := gcm.Open(nil, nonce[:], ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plaintext, nil
}

// EncryptChunk encrypts a chunk of plaintext for the given asset and chunk index.
func EncryptChunk(key []byte, assetID string, chunkIndex uint64, plaintext []byte) ([]byte, error) {
	nonce := ChunkNonce(chunkIndex)
	aad := ChunkAAD(assetID, chunkIndex)
	return Seal(key, nonce, plaintext, aad)
}

// DecryptChunk decrypts a chunk ciphertext for the given asset and chunk index.
func DecryptChunk(key []byte, assetID string, chunkIndex uint64, ciphertext []byte) ([]byte, error) {
	nonce := ChunkNonce(chunkIndex)
	aad := ChunkAAD(assetID, chunkIndex)
	return Open(key, nonce, ciphertext, aad)
}

// EncryptManifest encrypts the manifest JSON for the given asset.
func EncryptManifest(key []byte, assetID string, plaintext []byte) ([]byte, error) {
	nonce := ManifestNonce()
	aad := ManifestAAD(assetID)
	return Seal(key, nonce, plaintext, aad)
}

// DecryptManifest decrypts the manifest ciphertext for the given asset.
func DecryptManifest(key []byte, assetID string, ciphertext []byte) ([]byte, error) {
	nonce := ManifestNonce()
	aad := ManifestAAD(assetID)
	return Open(key, nonce, ciphertext, aad)
}
