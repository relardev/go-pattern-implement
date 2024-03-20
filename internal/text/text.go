package text

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

func ToDecl(packageText string) (ast.Decl, error) {
	template := `
	package abc

	{{TEXT}}
	`

	packageText = strings.Replace(template, "{{TEXT}}", packageText, 1)

	fset := token.NewFileSet()
	tree, err := parser.ParseFile(fset, "", packageText, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var found ast.Decl

	ast.Inspect(tree, func(node ast.Node) bool {
		switch typedNode := node.(type) {
		case ast.Decl:
			found = typedNode
			return false
		default:
			return true
		}
	})

	return found, nil
}

func ToStmts(packageText string) ([]ast.Stmt, error) {
	stmtsTemplate := `
	package abc
	
	func main() {
		{{TEXT}}
	}
	`
}
