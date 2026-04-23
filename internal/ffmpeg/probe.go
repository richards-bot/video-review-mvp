// Package ffmpeg provides wrappers around the ffmpeg and ffprobe binaries.
package ffmpeg

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// ProbeResult holds video metadata from ffprobe.
type ProbeResult struct {
	DurationMS int64
	Width      int
	Height     int
	Framerate  float64
	VideoCodec string
	AudioCodec string
}

type ffprobeOutput struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
	Streams []struct {
		CodecType      string `json:"codec_type"`
		CodecName      string `json:"codec_name"`
		Width          int    `json:"width"`
		Height         int    `json:"height"`
		RFrameRate     string `json:"r_frame_rate"`
		AvgFrameRate   string `json:"avg_frame_rate"`
		CodecTagString string `json:"codec_tag_string"`
	} `json:"streams"`
}

// Probe runs ffprobe on the given file and returns metadata.
func Probe(ctx context.Context, filePath string) (*ProbeResult, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("ffprobe failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("ffprobe: %w", err)
	}

	var probe ffprobeOutput
	if err := json.Unmarshal(out, &probe); err != nil {
		return nil, fmt.Errorf("parse ffprobe output: %w", err)
	}

	result := &ProbeResult{}

	// Parse duration
	if probe.Format.Duration != "" {
		secs, err := strconv.ParseFloat(probe.Format.Duration, 64)
		if err == nil {
			result.DurationMS = int64(secs * 1000)
		}
	}

	// Parse streams
	for _, s := range probe.Streams {
		switch s.CodecType {
		case "video":
			result.Width = s.Width
			result.Height = s.Height
			result.VideoCodec = s.CodecName
			result.Framerate = parseFramerate(s.AvgFrameRate)
			if result.Framerate == 0 {
				result.Framerate = parseFramerate(s.RFrameRate)
			}
		case "audio":
			result.AudioCodec = s.CodecName
		}
	}

	return result, nil
}

// CheckAvailable checks that ffmpeg and ffprobe are available on PATH.
func CheckAvailable() error {
	for _, bin := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("%s not found on PATH: install ffmpeg to use the uploader", bin)
		}
	}
	return nil
}

func parseFramerate(s string) float64 {
	parts := strings.Split(s, "/")
	if len(parts) == 2 {
		num, err1 := strconv.ParseFloat(parts[0], 64)
		den, err2 := strconv.ParseFloat(parts[1], 64)
		if err1 == nil && err2 == nil && den != 0 {
			return num / den
		}
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
