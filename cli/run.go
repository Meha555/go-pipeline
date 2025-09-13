package cli

import (
	"context"
	"fmt"

	"github.com/Meha555/go-pipeline/internal"
	"github.com/Meha555/go-pipeline/parser"
	"github.com/Meha555/go-pipeline/pipeline"

	"github.com/spf13/cobra"
)

// runCmd 执行pipeline
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a pipeline",
	Long:  "Run a pipeline through a config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		conf, err := parser.ParseConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("parsing %s failed: %w", configFile, err)
		}

		pipe := pipeline.MakePipeline(conf)

		ctx := context.Background()
		if verbose {
			ctx = context.WithValue(ctx, internal.VerboseKey, verbose)
		}

		status := pipe.Run(ctx)
		if status == pipeline.Failed {
			return fmt.Errorf("pipeline %s@%s run failed", pipe.Name, pipe.Version)
		}
		return nil
	},
}

var (
	configFile string
	verbose    bool
)

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output for jobs")
	runCmd.Flags().StringVarP(&configFile, "file", "f", "", "config file")
	runCmd.MarkFlagRequired("file")
}
