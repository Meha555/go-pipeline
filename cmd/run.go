package cmd

import (
	"go-pipeline/parser"
	"go-pipeline/pipeline"
	"log"

	"github.com/spf13/cobra"
)

// runCmd 执行pipeline
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the pipeline",
	Long:  "Run a pipeline through a config file",
	Run: func(cmd *cobra.Command, args []string) {
		pipe, err := parser.ParseConfigFile(configFile)
		if err != nil {
			log.Fatalf("parsing %s failed: %v", configFile, err)
		}
		log.Println("start running pipeline")
		if pipe.Run() == pipeline.Success {
			log.Println("pipeline run success")
		} else {
			log.Println("pipeline run failed")
		}
	},
}

var (
	configFile string
)

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVarP(&configFile, "file", "f", "", "config file")
	runCmd.MarkFlagRequired("file")
}
