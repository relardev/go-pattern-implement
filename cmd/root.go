package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pattern-implement",
	Short: "CLI utility to generate implementations of patterns",
	Long: `CLI utility to generate implementations of patterns.
Main idea is to generate the initial implementation, paste it 
into the project and then if needed, modify it.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().
		StringP("file", "f", "", "path to file with interface to implement")
}
