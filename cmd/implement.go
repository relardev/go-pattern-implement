package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"component-generator/internal/generator"

	"github.com/spf13/cobra"
)

var implementCmd = &cobra.Command{
	Use:   "implement <implementation>",
	Short: "Implemen an interface",
	Long: `Implemen an interface. This command will read stdin
and generate the implementation on stdout

	to find out available implementations, run:
	$ component-generator list
	`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		packageName, err := cmd.Flags().GetString("package")
		if err != nil {
			log.Fatal(err)
		}
		implementation := args[0]
		input := getTextFromStdin()
		g := generator.NewGenerator(true)
		g.Implement(input, implementation, packageName)
	},
}

func init() {
	rootCmd.AddCommand(implementCmd)
	implementCmd.Flags().
		StringP("package", "p", "", "package from which the interface comes from")
	err := implementCmd.MarkFlagRequired("package")
	if err != nil {
		log.Fatal(err)
	}
}

func getTextFromStdin() string {
	scanner := bufio.NewScanner(os.Stdin)

	var lines []string

	for scanner.Scan() {
		input := scanner.Text()
		lines = append(lines, input)
	}

	if scanner.Err() != nil {
		fmt.Println("Error:", scanner.Err())
	} else {
		_ = fmt.Errorf("Error: %s", scanner.Err())
	}

	return strings.Join(lines, "\n")
}
