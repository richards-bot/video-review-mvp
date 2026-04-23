package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/cobra"

	internalcrypto "github.com/guardian/video-review-mvp/internal/crypto"
	"github.com/guardian/video-review-mvp/internal/ffmpeg"
	"github.com/guardian/video-review-mvp/internal/manifest"
	"github.com/guardian/video-review-mvp/internal/s3upload"
	"github.com/guardian/video-review-mvp/internal/share"
)

const defaultChunkSize = 1024 * 1024 // 1 MiB
const defaultExpiryHours = 48

type uploadOptions struct {
	bucket       string
	region       string
	viewerURL    string
	author       string
	chunkSize    int
	crf          int
	maxBitrate   string
	keepProxy    bool
	outputFormat string
	verbose      bool
	expiryHours  int
}

func newUploadCommand() *cobra.Command {
	opts := &uploadOptions{}

	cmd := &cobra.Command{
		Use:   "upload <source-file>",
		Short: "Transcode, encrypt, and upload a video for review",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpload(cmd.Context(), args[0], opts)
		},
	}

	// Flags with env var fallbacks
	cmd.Flags().StringVar(&opts.bucket, "bucket", envOr("VIDEO_REVIEW_BUCKET", ""), "S3 bucket name (env: VIDEO_REVIEW_BUCKET)")
	cmd.Flags().StringVar(&opts.region, "region", envOr("VIDEO_REVIEW_REGION", envOr("AWS_REGION", "")), "AWS region (env: VIDEO_REVIEW_REGION, AWS_REGION)")
	cmd.Flags().StringVar(&opts.viewerURL, "viewer-url", envOr("VIDEO_REVIEW_VIEWER_URL", ""), "Viewer base URL (env: VIDEO_REVIEW_VIEWER_URL)")
	cmd.Flags().StringVar(&opts.author, "author", envOr("USER", "unknown"), "Uploader identifier (default: $USER)")
	cmd.Flags().IntVar(&opts.chunkSize, "chunk-size", defaultChunkSize, "Chunk size in bytes")
	cmd.Flags().IntVar(&opts.crf, "crf", 23, "ffmpeg x264 CRF quality (lower = better)")
	cmd.Flags().StringVar(&opts.maxBitrate, "max-bitrate", "5M", "ffmpeg maxrate")
	cmd.Flags().BoolVar(&opts.keepProxy, "keep-proxy", false, "Keep intermediate proxy file after upload")
	cmd.Flags().StringVar(&opts.outputFormat, "output-format", "url", "Output format: url or json")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "Verbose logging")
	cmd.Flags().IntVar(&opts.expiryHours, "expiry-hours", defaultExpiryHours, "Presigned URL expiry in hours")

	return cmd
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func runUpload(ctx context.Context, sourceFile string, opts *uploadOptions) error {
	// Set up logger
	level := slog.LevelInfo
	if opts.verbose {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	// ── Stage 1: Validate inputs ─────────────────────────────────────────────
	log.Debug("validating inputs")

	if _, err := os.Stat(sourceFile); err != nil {
		return fmt.Errorf("source file: %w", err)
	}

	if err := ffmpeg.CheckAvailable(); err != nil {
		return err
	}

	if opts.bucket == "" {
		return fmt.Errorf("--bucket is required (or set VIDEO_REVIEW_BUCKET)")
	}
	if opts.region == "" {
		return fmt.Errorf("--region is required (or set VIDEO_REVIEW_REGION)")
	}
	if opts.viewerURL == "" {
		return fmt.Errorf("--viewer-url is required (or set VIDEO_REVIEW_VIEWER_URL)")
	}
	if err := validateViewerURL(opts.viewerURL); err != nil {
		return err
	}

	log.Info("source file validated", "file", sourceFile)

	// ── Stage 2: Probe source ────────────────────────────────────────────────
	log.Info("probing source file")
	probeResult, err := ffmpeg.Probe(ctx, sourceFile)
	if err != nil {
		return fmt.Errorf("probe: %w", err)
	}
	log.Info("source probed",
		"duration_ms", probeResult.DurationMS,
		"width", probeResult.Width,
		"height", probeResult.Height,
		"framerate", probeResult.Framerate,
	)

	// ── Stage 3: Transcode ───────────────────────────────────────────────────
	tmpDir, err := os.MkdirTemp("", "video-review-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	if !opts.keepProxy {
		defer os.RemoveAll(tmpDir)
	}

	proxyPath := filepath.Join(tmpDir, "proxy.mp4")
	log.Info("transcoding to fragmented MP4", "output", proxyPath)
	transcodeStart := time.Now()

	transcodeOpts := ffmpeg.TranscodeOptions{
		CRF:        opts.crf,
		MaxBitrate: opts.maxBitrate,
		Verbose:    opts.verbose,
	}
	if err := ffmpeg.Transcode(ctx, sourceFile, proxyPath, transcodeOpts, log); err != nil {
		return fmt.Errorf("transcode: %w", err)
	}

	proxyInfo, err := os.Stat(proxyPath)
	if err != nil {
		return fmt.Errorf("stat proxy: %w", err)
	}
	log.Info("transcode complete",
		"elapsed", time.Since(transcodeStart).Round(time.Second),
		"size_bytes", proxyInfo.Size(),
	)

	// ── Stage 4: Generate keys and identifiers ───────────────────────────────
	cek := make([]byte, 32)
	if _, err := rand.Read(cek); err != nil {
		return fmt.Errorf("generate CEK: %w", err)
	}

	assetIDBytes := make([]byte, 16)
	if _, err := rand.Read(assetIDBytes); err != nil {
		return fmt.Errorf("generate asset_id: %w", err)
	}
	assetID := hex.EncodeToString(assetIDBytes)
	log.Debug("generated identifiers", "asset_id", assetID)

	// ── Stage 5: Compute plaintext SHA-256 ───────────────────────────────────
	proxyFile, err := os.Open(proxyPath)
	if err != nil {
		return fmt.Errorf("open proxy: %w", err)
	}

	hasher := sha256.New()
	totalBytes, err := io.Copy(hasher, proxyFile)
	if err != nil {
		proxyFile.Close()
		return fmt.Errorf("hash proxy: %w", err)
	}
	proxyFile.Close()
	plaintextSHA256 := hex.EncodeToString(hasher.Sum(nil))

	// ── Stage 6: Chunk, encrypt, upload ─────────────────────────────────────
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(opts.region))
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}
	s3Client := s3.NewFromConfig(awsCfg)
	presignClient := s3.NewPresignClient(s3Client)
	uploader := s3upload.New(s3Client, opts.bucket, log)
	expiry := time.Duration(opts.expiryHours) * time.Hour

	proxyFile, err = os.Open(proxyPath)
	if err != nil {
		return fmt.Errorf("open proxy for upload: %w", err)
	}
	defer proxyFile.Close()

	log.Info("starting upload", "asset_id", assetID, "total_bytes", totalBytes)

	var chunkRefs []manifest.ChunkRef
	var chunkIndex uint64
	lastProgressTime := time.Now()
	buf := make([]byte, opts.chunkSize)

	for {
		n, readErr := io.ReadFull(proxyFile, buf)
		if n > 0 {
			plaintext := buf[:n]
			ciphertext, err := internalcrypto.EncryptChunk(cek, assetID, chunkIndex, plaintext)
			if err != nil {
				return fmt.Errorf("encrypt chunk %d: %w (asset_id: %s)", chunkIndex, err, assetID)
			}

			key := fmt.Sprintf("%s/chunk_%06d.enc", assetID, chunkIndex)
			if err := uploader.PutObject(ctx, key, ciphertext); err != nil {
				return fmt.Errorf("upload chunk %d: %w (asset_id: %s)", chunkIndex, err, assetID)
			}

			// Generate presigned URL for this chunk
			presignURL, err := s3upload.PresignGetObject(ctx, presignClient, opts.bucket, key, expiry)
			if err != nil {
				return fmt.Errorf("presign chunk %d: %w (asset_id: %s)", chunkIndex, err, assetID)
			}

			chunkRefs = append(chunkRefs, manifest.ChunkRef{
				Index: int(chunkIndex),
				URL:   presignURL,
				Size:  int64(n),
			})

			// Progress reporting
			now := time.Now()
			if now.Sub(lastProgressTime) >= 5*time.Second {
				log.Info("upload progress", "chunks_uploaded", chunkIndex+1)
				lastProgressTime = now
			}

			chunkIndex++
		}

		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read proxy: %w (asset_id: %s)", readErr, assetID)
		}
	}

	log.Info("chunks uploaded", "chunk_count", chunkIndex, "asset_id", assetID)

	// ── Stage 7: Build and encrypt manifest ──────────────────────────────────
	videoCodec := buildVideoCodec(probeResult)
	audioCodec := buildAudioCodec(probeResult)

	m := &manifest.Manifest{
		Version:         manifest.CurrentVersion,
		AssetID:         assetID,
		CreatedAt:       time.Now().UTC(),
		CreatedBy:       opts.author,
		SourceFilename:  filepath.Base(sourceFile),
		Codec:           manifest.Codec{Video: videoCodec, Audio: audioCodec, MIME: buildMIME(videoCodec, audioCodec)},
		DurationMS:      probeResult.DurationMS,
		Width:           probeResult.Width,
		Height:          probeResult.Height,
		Framerate:       probeResult.Framerate,
		ChunkSize:       opts.chunkSize,
		ChunkCount:      int(chunkIndex),
		TotalBytes:      totalBytes,
		PlaintextSHA256: plaintextSHA256,
		Chunks:          chunkRefs,
		ExpiresAt:       time.Now().UTC().Add(expiry),
	}

	manifestJSON, err := manifest.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w (asset_id: %s)", err, assetID)
	}

	manifestCiphertext, err := internalcrypto.EncryptManifest(cek, assetID, manifestJSON)
	if err != nil {
		return fmt.Errorf("encrypt manifest: %w (asset_id: %s)", err, assetID)
	}

	manifestKey := assetID + "/manifest.enc"
	if err := uploader.PutObject(ctx, manifestKey, manifestCiphertext); err != nil {
		return fmt.Errorf("upload manifest: %w (asset_id: %s)", err, assetID)
	}

	// ── Stage 8: Emit share URL ──────────────────────────────────────────────
	shareURL, err := share.BuildShareURL(opts.viewerURL, assetID, cek, opts.bucket, opts.region)
	if err != nil {
		return fmt.Errorf("build share URL: %w", err)
	}

	switch opts.outputFormat {
	case "json":
		out := map[string]interface{}{
			"asset_id":    assetID,
			"bucket":      opts.bucket,
			"region":      opts.region,
			"viewer_url":  opts.viewerURL,
			"share_url":   shareURL,
			"chunk_count": int(chunkIndex),
			"duration_ms": probeResult.DurationMS,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			return fmt.Errorf("encode JSON output: %w", err)
		}
	default:
		fmt.Println(shareURL)
	}

	log.Info("upload complete", "asset_id", assetID, "chunk_count", chunkIndex)
	return nil
}

func validateViewerURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid viewer URL: %w", err)
	}
	if u.Scheme == "https" {
		return nil
	}
	if u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1") {
		return nil
	}
	return fmt.Errorf("viewer URL must use https:// (http://localhost is allowed for dev), got: %s", rawURL)
}

func buildVideoCodec(p *ffmpeg.ProbeResult) string {
	// Return H.264 codec string — the proxy is always transcoded to H.264 main profile
	return "avc1.64001f"
}

func buildAudioCodec(p *ffmpeg.ProbeResult) string {
	return "mp4a.40.2"
}

func buildMIME(videoCodec, audioCodec string) string {
	return fmt.Sprintf(`video/mp4; codecs="%s, %s"`, videoCodec, audioCodec)
}

// formatDuration formats milliseconds as mm:ss for display.
func formatDuration(ms int64) string {
	secs := ms / 1000
	mins := secs / 60
	secs = secs % 60
	return fmt.Sprintf("%d:%02d", mins, secs)
}

// isoMIME returns the ISO MIME for ffmpeg-produced H.264/AAC fMP4.
// The exact codec string depends on ffprobe output; for the fixed
// transcode parameters in Stage 3 the profile is always Main@4.0.
// In production, parse this from ffprobe output on the proxy file.
func isoMIME(video, audio string) string {
	return fmt.Sprintf(`video/mp4; codecs="%s, %s"`, video, audio)
}

var _ = strings.Contains // keep import
var _ = formatDuration   // keep for future use
var _ = isoMIME          // keep for future use
