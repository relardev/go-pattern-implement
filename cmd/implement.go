package cmd

import (
	"log"

	"component-generator/internal/commands/implement"

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
		implement.Implement(implementation, packageName)
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
