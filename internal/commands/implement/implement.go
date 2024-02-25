package implement

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"component-generator/internal/implementations/prometheus"
	slogImp "component-generator/internal/implementations/slog"
)

const mainTemplate = `
package whatever

{{TEXT}}

`

func Implement(implementation, packageName string) {
	text := getTextFromStdin()

	filledTemplate := strings.Replace(mainTemplate, "{{TEXT}}", text, 1)

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "main.go", filledTemplate, 0)
	if err != nil {
		panic(err)
	}

	var visitor func(node ast.Node) bool

	switch implementation {
	case "prometheus":
		visitor = prometheus.Visitor(packageName, fset)
	case "slog":
		visitor = slogImp.Visitor(packageName, fset)
	default:
		fmt.Println("Unknown implementation", implementation)
		os.Exit(1)
	}

	ast.Inspect(f, visitor)
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
