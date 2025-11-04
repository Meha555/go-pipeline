package cli

import (
	"os"

	"github.com/Meha555/go-pipeline/internal"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "go-pipeline",
	Short:   "A tool to run workflow",
	Version: internal.Version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SilenceUsage = true
	rootCmd.SetVersionTemplate("{{.Version}}")
}
