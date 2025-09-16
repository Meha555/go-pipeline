package cli

import (
	"fmt"

	"github.com/Meha555/go-pipeline/pipeline"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "envs",
	Short: "Print all built-in environment variables",
	Run: func(cmd *cobra.Command, args []string) {
		for _, env := range pipeline.Builtins {
			fmt.Printf("%-18s - %s\n", env.Name, env.Description)
		}
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
}
