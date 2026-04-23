// Package share handles share URL construction and CEK encoding.
package share

import (
	"encoding/base64"
	"fmt"
	"net/url"
)

// EncodeCEK encodes the CEK as URL-safe base64 without padding (RFC 4648 §5).
func EncodeCEK(cek []byte) string {
	return base64.RawURLEncoding.EncodeToString(cek)
}

// DecodeCEK decodes a URL-safe base64 CEK string.
func DecodeCEK(s string) ([]byte, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("decode CEK: %w", err)
	}
	if len(b) != 32 {
		return nil, fmt.Errorf("CEK must be 32 bytes, got %d", len(b))
	}
	return b, nil
}

// BuildShareURL constructs the share URL per §4.4 of the spec.
// fragment: #v=1&a=<asset_id>&k=<cek_b64url>&b=<bucket>&r=<region>
func BuildShareURL(viewerURL, assetID string, cek []byte, bucket, region string) (string, error) {
	base, err := url.Parse(viewerURL)
	if err != nil {
		return "", fmt.Errorf("parse viewer URL: %w", err)
	}
	// Ensure path is set so URL renders as <host>/#fragment not <host>#fragment
	if base.Path == "" {
		base.Path = "/"
	}
	fragment := fmt.Sprintf("v=1&a=%s&k=%s&b=%s&r=%s",
		assetID,
		EncodeCEK(cek),
		bucket,
		region,
	)
	base.Fragment = fragment
	return base.String(), nil
}
