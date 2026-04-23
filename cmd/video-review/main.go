package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	root := &cobra.Command{
		Use:   "video-review",
		Short: "Secure video upload and review tool",
		Long: `video-review encrypts and uploads video content to S3 for secure browser review.
Content is encrypted client-side; S3 never holds decryption keys.`,
		Version: fmt.Sprintf("%s (built %s)", version, buildTime),
	}

	root.AddCommand(newUploadCommand())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
