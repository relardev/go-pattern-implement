package implement

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"

	filegetter "component-generator/internal/implementations/file_getter"
	"component-generator/internal/implementations/prometheus"
	slogImp "component-generator/internal/implementations/slog"
)

var templates = []string{
	`
package whatever

{{TEXT}}

`,
	`
package whatever

type xxx {{TEXT}}
`,
}

func Implement(implementation, packageName string) {
	text := getTextFromStdin()

	fset := token.NewFileSet()

	var parsed *ast.File
	var err error

	for _, template := range templates {
		filledTemplate := strings.Replace(template, "{{TEXT}}", text, 1)

		parsed, err = parser.ParseFile(fset, "main.go", filledTemplate, 0)
		if err == nil {
			break
		}
	}

	if err != nil {
		log.Fatalf("None of the themplates parsed, last error: %s", err)
	}

	var visitor func(node ast.Node) bool

	switch implementation {
	case "prometheus":
		visitor = prometheus.Visitor(packageName, fset)
	case "slog":
		visitor = slogImp.Visitor(packageName, fset)
	case "filegetter":
		visitor = filegetter.Visitor(packageName, fset)
	default:
		fmt.Println("Unknown implementation", implementation)
		os.Exit(1)
	}

	ast.Inspect(parsed, visitor)
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
