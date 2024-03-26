package generator

import (
	"component-generator/internal/implementations/cache"
	"component-generator/internal/implementations/metrics"
	"component-generator/internal/implementations/semaphore"
	"component-generator/internal/implementations/slog"
	"component-generator/internal/implementations/store"
	"component-generator/internal/implementations/throttle"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"strings"

	filegetter "component-generator/internal/implementations/file_getter"
)

type template string

const (
	interfaceTempl template = `
package whatever

{{TEXT}}

`

	funcTypeTempl template = `
package whatever

type xxx {{TEXT}}
`
)

var templates = []template{
	interfaceTempl,
	funcTypeTempl,
}

type implementator interface {
	Visit(node ast.Node) (bool, []ast.Decl)
	Name() string
	Description() string
	Error() error
}

type Generator struct {
	printResult bool
}

func NewGenerator(printResult bool) *Generator {
	return &Generator{printResult: printResult}
}

func (g *Generator) ListAvailableImplementators(input string) ([]string, error) {
	fset := token.NewFileSet()

	implementators := g.implementators("aaa")

	list := make([]implementator, 0)

	for _, possible := range implementators {
		var parsed *ast.File

		var err error

		for _, template := range templates {
			filledTemplate := strings.Replace(string(template), "{{TEXT}}", input, 1)

			parsed, err = parser.ParseFile(fset, "main.go", filledTemplate, parser.ParseComments)
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

			if possible.Error() != nil {
				return
			}

			list = append(list, possible)
		}
		recoverable()
	}

	return g.stringRepresentations(list), nil
}

func (g *Generator) Implement(input, implementation, packageName string) {
	fset := token.NewFileSet()

	var parsed *ast.File

	var err error

	for _, template := range templates {
		filledTemplate := strings.Replace(string(template), "{{TEXT}}", input, 1)

		parsed, err = parser.ParseFile(fset, "main.go", filledTemplate, parser.ParseComments)
		if err == nil {
			break
		}
	}

	if err != nil {
		log.Fatalf("None of the templates parsed, last error: %s", err)
	}

	var visitor func(ast.Node) (bool, []ast.Decl)

	implementators := g.implementators(packageName)

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

func (g *Generator) ListAllImplementators() []string {
	all := g.implementators("aaa")
	return g.stringRepresentations(all)
}

func (g *Generator) stringRepresentations(impl []implementator) []string {
	names := make([]string, 0, len(impl))

	for _, i := range impl {
		names = append(names, i.Name()+" - "+i.Description())
	}

	return names
}

func (g *Generator) implementators(packageName string) []implementator {
	return []implementator{
		metrics.New(packageName, "prometheus", "prometheus"),
		metrics.New(packageName, "statsd", "statsd"),
		slog.New(packageName),
		filegetter.New(packageName),
		store.New(packageName, store.PanicInNew),
		store.New(packageName, store.ReturnErrorInNew),
		cache.New(packageName),
		semaphore.New(packageName),
		throttle.New(packageName, throttle.ModeNoError),
		throttle.New(packageName, throttle.ModeWithError),
	}
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
				fmt.Fprint(os.Stdout, "\n")
			}

			return false
		}

		return true
	}
}
