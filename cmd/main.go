package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"

	"component-generator/internal/implementations/prometheus"
	slogImp "component-generator/internal/implementations/slog"
)

const mainTemplate = `
package xxx

type Repo interface {
	Find(id int) (domain.User, error)
	FindBy(id1, id2 int) (domain.User, error)
	FindByInt(int, int) (domain.User, error)
	FindAll() ([]domain.User, error)
	Save(domain.User) error
	ApplyToAll(func(domain.User) error)
}

`

func main() {
	packageName := flag.String("package", "xxx", "package from which inteface comes from")

	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		fmt.Println("No positional argument provided")
		os.Exit(1)
	}

	implementation := args[0]

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "main.go", mainTemplate, 0)
	if err != nil {
		panic(err)
	}

	var visitor func(node ast.Node) bool

	switch implementation {
	case "prometheus":
		visitor = prometheus.Visitor(*packageName, fset)
	case "slog":
		visitor = slogImp.Visitor(*packageName, fset)
	default:
		fmt.Println("Unknown implementation", implementation)
		os.Exit(1)
	}

	ast.Inspect(f, visitor)
}
