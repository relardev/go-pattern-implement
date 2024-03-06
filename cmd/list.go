package cmd

import (
	"fmt"
	"log"
	"strings"

	"component-generator/internal/generator"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List implementations",
	Long:  `List implementations.`,
	Run: func(cmd *cobra.Command, args []string) {
		available, err := cmd.Flags().GetBool("available")
		if err != nil {
			log.Fatal(err)
		}

		filePath, err := cmd.Flags().GetString("file")
		if err != nil {
			log.Fatal(err)
		}

		var list []string

		g := generator.NewGenerator(false)
		if available {
			list, err = g.ListAvailableImplementators(
				getInput(filePath),
			)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			list = g.ListAllImplementators()
		}
		final := strings.Join(list, "\n")
		fmt.Printf("%v", final+"\n")
	},
}

func init() {
	listCmd.Flags().
		BoolP(
			"available",
			"a",
			false,
			"List only available implementations based on stdin or file.",
		)
	rootCmd.AddCommand(listCmd)
}
