package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"strings"

	filegetter "component-generator/internal/implementations/file_getter"
	"component-generator/internal/implementations/prometheus"
)

// TODO maybe generate the list from implementations?
var Implementations = []string{
	"prometheus",
	"filegetter",
}

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

type implementator interface {
	Visit(node ast.Node) (bool, []ast.Decl)
	Name() string
	Error() error
}

type Generator struct {
	printResult bool
}

func NewGenerator(printResult bool) *Generator {
	return &Generator{printResult: printResult}
}

func (g *Generator) GetAvailableImplementations(input string) ([]string, error) {
	packageName := "aaa"
	fset := token.NewFileSet()

	implementator := []implementator{
		prometheus.New(packageName),
		filegetter.New(packageName),
	}

	list := make([]string, 0)

	for _, possible := range implementator {
		var parsed *ast.File

		var err error

		for _, template := range templates {
			filledTemplate := strings.Replace(template, "{{TEXT}}", input, 1)

			parsed, err = parser.ParseFile(fset, "main.go", filledTemplate, 0)
			if err == nil {
				break
			}
		}

		if err != nil {
			log.Fatalf("None of the themplates parsed, last error: %s", err)
		}

		wrappedVisitor := g.wrap(possible.Visit)
		recoverable := func() {
			defer func() {
				_ = recover()
			}()
			ast.Inspect(parsed, wrappedVisitor)
			list = append(list, possible.Name())
		}
		recoverable()
	}

	return list, nil
}

func (g *Generator) Implement(input, implementation, packageName string) {
	fset := token.NewFileSet()

	var parsed *ast.File

	var err error

	for _, template := range templates {
		filledTemplate := strings.Replace(template, "{{TEXT}}", input, 1)

		parsed, err = parser.ParseFile(fset, "main.go", filledTemplate, 0)
		if err == nil {
			break
		}
	}

	if err != nil {
		log.Fatalf("None of the themplates parsed, last error: %s", err)
	}

	implementators := []implementator{
		prometheus.New(packageName),
		filegetter.New(packageName),
	}

	var visitor func(ast.Node) (bool, []ast.Decl)

	for _, possible := range implementators {
		if possible.Name() == implementation {
			visitor = possible.Visit
			break
		}
	}

	if visitor == nil {
		fmt.Println("Unknown implementation", implementation)
		os.Exit(1)
	}

	wrappedVisitor := g.wrap(visitor)

	ast.Inspect(parsed, wrappedVisitor)
}

func (g *Generator) wrap(
	visitor func(ast.Node) (bool, []ast.Decl),
) func(ast.Node) bool {
	return func(node ast.Node) bool {
		if node == nil {
			return true
		}

		keepGoing, decls := visitor(node)
		if !keepGoing {
			if g.printResult {
				printer.Fprint(os.Stdout, token.NewFileSet(), decls)
			}

			return false
		}

		return true
	}
}
