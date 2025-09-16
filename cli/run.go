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
		// 没有被注册到cobra的参数会被认为是额外参数出现在这里的args中
		conf, err := parser.ParseConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("parsing %s failed: %w", configFile, err)
		}

		pipe := pipeline.MakePipeline(conf)

		ctx := context.Background()
		// 处理额外的参数
		parser.ParseArgs(args, ctx)
		if verbose {
			ctx = context.WithValue(ctx, internal.VerboseKey, verbose)
		}
		if trace && !dryRun {
			ctx = context.WithValue(ctx, internal.TraceKey, trace)
		}
		if dryRun {
			ctx = context.WithValue(ctx, internal.DryRunKey, dryRun)
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
	trace      bool
	dryRun     bool
)

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output for jobs")
	runCmd.Flags().BoolVarP(&trace, "trace", "t", false, "time trace for jobs")
	runCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "dry run")
	runCmd.Flags().StringVarP(&configFile, "file", "f", "", "config file")
	runCmd.MarkFlagRequired("file")
}
