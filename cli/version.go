package cli

import (
	"fmt"

	"github.com/Meha555/go-pipeline/internal"
	"github.com/spf13/cobra"
)

// versionCmd 打印版本号
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Version",
	Long:  "Print Version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(internal.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
