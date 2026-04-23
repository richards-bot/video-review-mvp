package manifest

import (
	"strings"
	"testing"
	"time"
)

func TestMarshalUnmarshal_RoundTrip(t *testing.T) {
	now := time.Date(2026, 4, 23, 14, 30, 0, 0, time.UTC)
	expires := now.Add(48 * time.Hour)

	m := &Manifest{
		Version:         CurrentVersion,
		AssetID:         "7f3a9c2e4b1d8f600123456789abcdef",
		CreatedAt:       now,
		CreatedBy:       "rich@example.com",
		SourceFilename:  "interview.mov",
		Codec:           Codec{Video: "avc1.64001f", Audio: "mp4a.40.2", MIME: `video/mp4; codecs="avc1.64001f, mp4a.40.2"`},
		DurationMS:      127500,
		Width:           1920,
		Height:          1080,
		Framerate:       25.0,
		ChunkSize:       1048576,
		ChunkCount:      2,
		TotalBytes:      2000000,
		PlaintextSHA256: "abc123",
		Chunks: []ChunkRef{
			{Index: 0, URL: "https://example.com/chunk0", Size: 1048576},
			{Index: 1, URL: "https://example.com/chunk1", Size: 951424},
		},
		ExpiresAt: expires,
	}

	data, err := Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	m2, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if m2.AssetID != m.AssetID {
		t.Errorf("AssetID: got %q, want %q", m2.AssetID, m.AssetID)
	}
	if m2.Version != m.Version {
		t.Errorf("Version: got %d, want %d", m2.Version, m.Version)
	}
	if m2.ChunkCount != m.ChunkCount {
		t.Errorf("ChunkCount: got %d, want %d", m2.ChunkCount, m.ChunkCount)
	}
	if len(m2.Chunks) != len(m.Chunks) {
		t.Errorf("Chunks len: got %d, want %d", len(m2.Chunks), len(m.Chunks))
	}
}

func TestUnmarshal_WrongVersion(t *testing.T) {
	data := []byte(`{"version":2,"asset_id":"7f3a9c2e4b1d8f600123456789abcdef","chunk_size":1048576,"chunk_count":0}`)
	_, err := Unmarshal(data)
	if err == nil {
		t.Error("expected error for wrong version, got nil")
	}
}

func TestUnmarshal_InvalidAssetID(t *testing.T) {
	data := []byte(`{"version":1,"asset_id":"short","chunk_size":1048576,"chunk_count":0}`)
	_, err := Unmarshal(data)
	if err == nil {
		t.Error("expected error for invalid asset_id, got nil")
	}
}

func TestMarshal_Compact(t *testing.T) {
	m := &Manifest{
		Version:    CurrentVersion,
		AssetID:    "7f3a9c2e4b1d8f600123456789abcdef",
		ChunkSize:  1048576,
		ChunkCount: 0,
	}
	data, err := Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	// Compact JSON should not have newlines
	if strings.Contains(string(data), "\n") {
		t.Error("Marshal output contains newlines (not compact)")
	}
}
