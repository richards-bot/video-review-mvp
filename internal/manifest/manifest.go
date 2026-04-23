// Package manifest defines the manifest schema and serialisation for video review assets.
package manifest

import (
	"encoding/json"
	"fmt"
	"time"
)

// Codec holds the codec identifiers for the proxy video.
type Codec struct {
	Video string `json:"video"`
	Audio string `json:"audio"`
	MIME  string `json:"mime"`
}

// ChunkRef holds a chunk index, presigned URL, and size for the viewer.
type ChunkRef struct {
	Index int    `json:"index"`
	URL   string `json:"url"`
	Size  int64  `json:"size"`
}

// Manifest is the per-asset metadata stored encrypted in S3.
type Manifest struct {
	Version         int        `json:"version"`
	AssetID         string     `json:"asset_id"`
	CreatedAt       time.Time  `json:"created_at"`
	CreatedBy       string     `json:"created_by"`
	SourceFilename  string     `json:"source_filename"`
	Codec           Codec      `json:"codec"`
	DurationMS      int64      `json:"duration_ms"`
	Width           int        `json:"width"`
	Height          int        `json:"height"`
	Framerate       float64    `json:"framerate"`
	ChunkSize       int        `json:"chunk_size"`
	ChunkCount      int        `json:"chunk_count"`
	TotalBytes      int64      `json:"total_bytes"`
	PlaintextSHA256 string     `json:"plaintext_sha256"`
	Chunks          []ChunkRef `json:"chunks"`
	ExpiresAt       time.Time  `json:"expires_at"`
}

// CurrentVersion is the manifest protocol version this code produces.
const CurrentVersion = 1

// Marshal serialises the manifest to compact JSON.
func Marshal(m *Manifest) ([]byte, error) {
	return json.Marshal(m)
}

// Unmarshal deserialises the manifest from JSON.
func Unmarshal(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("unmarshal manifest: %w", err)
	}
	if err := validate(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

func validate(m *Manifest) error {
	if m.Version != CurrentVersion {
		return fmt.Errorf("unsupported manifest version %d (expected %d)", m.Version, CurrentVersion)
	}
	if len(m.AssetID) != 32 {
		return fmt.Errorf("invalid asset_id length %d (expected 32)", len(m.AssetID))
	}
	if m.ChunkCount < 0 {
		return fmt.Errorf("invalid chunk_count %d", m.ChunkCount)
	}
	if m.ChunkSize <= 0 {
		return fmt.Errorf("invalid chunk_size %d", m.ChunkSize)
	}
	return nil
}
