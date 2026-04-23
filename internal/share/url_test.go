package share

import (
	"bytes"
	"strings"
	"testing"
)

func TestEncodeDecode_RoundTrip(t *testing.T) {
	cek := bytes.Repeat([]byte{0xAB}, 32)
	encoded := EncodeCEK(cek)
	if len(encoded) != 43 {
		t.Errorf("encoded length = %d, want 43", len(encoded))
	}
	decoded, err := DecodeCEK(encoded)
	if err != nil {
		t.Fatalf("DecodeCEK: %v", err)
	}
	if !bytes.Equal(decoded, cek) {
		t.Error("decoded CEK differs from original")
	}
}

func TestDecodeCEK_WrongLength(t *testing.T) {
	// 22 base64url chars = 16 bytes (not 32) - should fail validation
	shortEncoded := "AAAAAAAAAAAAAAAAAAAAAA"
	_, err := DecodeCEK(shortEncoded)
	if err == nil {
		t.Error("expected error for short CEK, got nil")
	}
}

func TestBuildShareURL(t *testing.T) {
	cek := bytes.Repeat([]byte{0x00}, 32)
	assetID := "7f3a9c2e4b1d8f600123456789abcdef"
	shareURL, err := BuildShareURL("https://review.example.com", assetID, cek, "my-bucket", "eu-west-2")
	if err != nil {
		t.Fatalf("BuildShareURL: %v", err)
	}
	if !strings.HasPrefix(shareURL, "https://review.example.com/#") {
		t.Errorf("URL missing expected prefix: %s", shareURL)
	}
	if !strings.Contains(shareURL, "v=1") {
		t.Error("URL missing v=1")
	}
	if !strings.Contains(shareURL, "a="+assetID) {
		t.Error("URL missing asset_id")
	}
	if !strings.Contains(shareURL, "b=my-bucket") {
		t.Error("URL missing bucket")
	}
	if !strings.Contains(shareURL, "r=eu-west-2") {
		t.Error("URL missing region")
	}
}
