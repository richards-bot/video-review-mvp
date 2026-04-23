package ffmpeg

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
)

// TranscodeOptions configures the ffmpeg transcode.
type TranscodeOptions struct {
	CRF        int
	MaxBitrate string
	Verbose    bool
}

// DefaultTranscodeOptions returns sensible defaults.
func DefaultTranscodeOptions() TranscodeOptions {
	return TranscodeOptions{
		CRF:        23,
		MaxBitrate: "5M",
	}
}

// Transcode converts the input file to a fragmented MP4 at outputPath.
func Transcode(ctx context.Context, inputPath, outputPath string, opts TranscodeOptions, log *slog.Logger) error {
	bufsize := multiplyBitrate(opts.MaxBitrate, 2)

	args := []string{
		"-i", inputPath,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", strconv.Itoa(opts.CRF),
		"-maxrate", opts.MaxBitrate,
		"-bufsize", bufsize,
		"-pix_fmt", "yuv420p",
		"-profile:v", "main",
		"-level", "4.0",
		"-c:a", "aac",
		"-b:a", "128k",
		"-ac", "2",
		"-movflags", "+frag_keyframe+empty_moov+default_base_moof+faststart",
		"-frag_duration", "2000000",
		"-f", "mp4",
		"-y",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	if opts.Verbose {
		log.Debug("ffmpeg command", "args", args)
		cmd.Stderr = &slogWriter{log: log}
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ffmpeg transcode failed: %w", err)
		}
		return nil
	}

	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg transcode failed: %s", stderr.String())
	}
	return nil
}

// multiplyBitrate doubles a bitrate string like "5M" -> "10M".
func multiplyBitrate(s string, factor int) string {
	if len(s) == 0 {
		return s
	}
	suffix := s[len(s)-1]
	if suffix >= '0' && suffix <= '9' {
		v, err := strconv.Atoi(s)
		if err == nil {
			return strconv.Itoa(v * factor)
		}
		return s
	}
	v, err := strconv.Atoi(s[:len(s)-1])
	if err != nil {
		return s
	}
	return strconv.Itoa(v*factor) + string(suffix)
}

// slogWriter writes lines to slog debug.
type slogWriter struct {
	log *slog.Logger
}

func (w *slogWriter) Write(p []byte) (int, error) {
	w.log.Debug("ffmpeg", "output", string(p))
	return len(p), nil
}
