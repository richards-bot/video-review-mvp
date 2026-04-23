// gen-fixtures generates the deterministic test fixture files for viewer/fixtures/.
// Run: go run ./cmd/gen-fixtures/
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	internalcrypto "github.com/guardian/video-review-mvp/internal/crypto"
	"github.com/guardian/video-review-mvp/internal/manifest"
)

func main() {
	// Fixed test parameters — deterministic
	cekHex := "0000000000000000000000000000000000000000000000000000000000000000"
	assetID := "aabbccddeeff00112233445566778899"
	cek, _ := hex.DecodeString(cekHex)

	// Fixed chunk plaintexts
	chunkPlaintexts := [][]byte{
		[]byte("chunk0data"),
		[]byte("chunk1data"),
		[]byte("chunk2data"),
	}

	outDir := filepath.Join("viewer", "fixtures")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fatalf("mkdir: %v", err)
	}

	// Encrypt and write chunks
	var chunkRefs []manifest.ChunkRef
	for i, pt := range chunkPlaintexts {
		ct, err := internalcrypto.EncryptChunk(cek, assetID, uint64(i), pt)
		if err != nil {
			fatalf("encrypt chunk %d: %v", i, err)
		}
		fname := fmt.Sprintf("fixture-chunk-%06d.enc", i)
		path := filepath.Join(outDir, fname)
		if err := os.WriteFile(path, ct, 0644); err != nil {
			fatalf("write %s: %v", path, err)
		}
		fmt.Printf("wrote %s (%d bytes)\n", path, len(ct))

		chunkRefs = append(chunkRefs, manifest.ChunkRef{
			Index: i,
			URL:   fmt.Sprintf("fixture://chunk%d", i),
			Size:  int64(len(pt)),
		})
	}

	// Build and encrypt manifest
	m := &manifest.Manifest{
		Version:         manifest.CurrentVersion,
		AssetID:         assetID,
		CreatedAt:       time.Date(2026, 4, 23, 14, 30, 0, 0, time.UTC),
		CreatedBy:       "fixture@test.local",
		SourceFilename:  "fixture.mp4",
		Codec:           manifest.Codec{Video: "avc1.64001f", Audio: "mp4a.40.2", MIME: `video/mp4; codecs="avc1.64001f, mp4a.40.2"`},
		DurationMS:      5000,
		Width:           320,
		Height:          240,
		Framerate:       25.0,
		ChunkSize:       10,
		ChunkCount:      3,
		TotalBytes:      30,
		PlaintextSHA256: "placeholder",
		Chunks:          chunkRefs,
		ExpiresAt:       time.Date(2026, 4, 25, 14, 30, 0, 0, time.UTC),
	}

	manifestJSON, err := manifest.Marshal(m)
	if err != nil {
		fatalf("marshal manifest: %v", err)
	}

	manifestCT, err := internalcrypto.EncryptManifest(cek, assetID, manifestJSON)
	if err != nil {
		fatalf("encrypt manifest: %v", err)
	}

	manifestPath := filepath.Join(outDir, "fixture-manifest.enc")
	if err := os.WriteFile(manifestPath, manifestCT, 0644); err != nil {
		fatalf("write manifest: %v", err)
	}
	fmt.Printf("wrote %s (%d bytes)\n", manifestPath, len(manifestCT))

	// Write fixture.json with all parameters
	fixtureJSON := map[string]interface{}{
		"note":        "Test fixture for viewer unit tests. CEK is all-zeros (32 bytes). Asset ID is fixed.",
		"asset_id":    assetID,
		"cek_hex":     cekHex,
		"cek_b64url":  "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"chunk_count": 3,
		"chunk_plaintexts": []string{
			string(chunkPlaintexts[0]),
			string(chunkPlaintexts[1]),
			string(chunkPlaintexts[2]),
		},
		"manifest_plaintext_json": string(manifestJSON),
		"files": map[string]string{
			"manifest": "fixture-manifest.enc",
			"chunk_0":  "fixture-chunk-000000.enc",
			"chunk_1":  "fixture-chunk-000001.enc",
			"chunk_2":  "fixture-chunk-000002.enc",
		},
	}

	fixtureJSONBytes, err := json.MarshalIndent(fixtureJSON, "", "  ")
	if err != nil {
		fatalf("marshal fixture.json: %v", err)
	}

	fixturePath := filepath.Join(outDir, "fixture.json")
	if err := os.WriteFile(fixturePath, fixtureJSONBytes, 0644); err != nil {
		fatalf("write fixture.json: %v", err)
	}
	fmt.Printf("wrote %s\n", fixturePath)
	fmt.Println("done")
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "gen-fixtures: "+format+"\n", args...)
	os.Exit(1)
}
