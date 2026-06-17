package cli

import (
	"io"
	"os"

	"github.com/Meha555/go-pipeline/internal"
	"github.com/Meha555/go-pipeline/internal/logging"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "go-pipeline",
	Short:   "A tool to run workflow",
	Version: internal.ResolveVersion(internal.BuildVersion()),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return configureLogging(logFormat, flagChanged(cmd, "log-format"), logLevel, flagChanged(cmd, "log-level"), logColor, flagChanged(cmd, "log-color"))
	},
}

var (
	logFormat     string
	logLevel      string
	logColor      string
	loggingWriter io.Writer = os.Stderr
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func configureLogging(format string, hasFormat bool, level string, hasLevel bool, color string, hasColor bool) error {
	opts := logging.ResolveOptions(format, hasFormat, level, hasLevel, color, hasColor)
	opts.Writer = loggingWriter
	return logging.Configure(opts)
}

func flagChanged(cmd *cobra.Command, name string) bool {
	if flag := cmd.Flags().Lookup(name); flag != nil {
		return flag.Changed
	}
	if flag := cmd.InheritedFlags().Lookup(name); flag != nil {
		return flag.Changed
	}
	return false
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SilenceUsage = true
	rootCmd.SetVersionTemplate("{{.Version}}")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", logging.FormatConsole, "log format: {console|json}")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", logging.LevelInfo, "log level: {debug|info|warn|error|disabled}")
	rootCmd.PersistentFlags().StringVar(&logColor, "log-color", logging.ColorAuto, "log color: {auto|never}")
}
