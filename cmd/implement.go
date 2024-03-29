package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/relardev/go-pattern-implement/internal/generator"

	"github.com/spf13/cobra"
)

var implementCmd = &cobra.Command{
	Use:   "implement <implementation>",
	Short: "Implemen an interface",
	Long: `Implemen an interface. This command will read stdin or file
and generate the implementation on stdout

	to find out available implementations, run:
	$ pattern-implement list
	`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		packageName, err := cmd.Flags().GetString("package")
		if err != nil {
			log.Fatal(err)
		}

		filePath, err := cmd.Flags().GetString("file")
		if err != nil {
			log.Fatal(err)
		}

		input := getInput(filePath)

		implementation := args[0]
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

func getInput(filePath string) string {
	if filePath == "" {
		return getTextFromStdin()
	}

	file, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal(err)
	}

	return string(file)
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
