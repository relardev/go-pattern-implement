package cmd

import (
	"fmt"

	"component-generator/internal/commands/list"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available implementations",
	Long:  `List available implementations.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%v", list.List())
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
