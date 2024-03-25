package text

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// ToDecl converts a string to a declaration
// Warning: This function panics, use only in code that recovers
func ToDecl(input string) ast.Decl {
	template := `
	package abc

	{{TEXT}}
	`

	packageText := strings.Replace(template, "{{TEXT}}", input, 1)

	fset := token.NewFileSet()
	tree, err := parser.ParseFile(fset, "", packageText, parser.ParseComments)
	if err != nil {
		panic(err)
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
	if found == nil {
		panic("decl not found")
	}

	return found
}

// ToDecl converts a string to list of statements
// Warning: This function panics, use only in code that recovers
func ToStmts(packageText string) []ast.Stmt {
	stmtsTemplate := `
	package abc
	
	func main() {
		{{TEXT}}
	}
	`

	packageText = strings.Replace(stmtsTemplate, "{{TEXT}}", packageText, 1)

	fset := token.NewFileSet()
	tree, err := parser.ParseFile(fset, "", packageText, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	var found []ast.Stmt

	ast.Inspect(tree, func(node ast.Node) bool {
		switch typedNode := node.(type) {
		case *ast.BlockStmt:
			found = typedNode.List
			return false
		default:
			return true
		}
	})

	if found == nil {
		panic("stmts not found")
	}

	return found
}

func ToExpr(input string) ast.Expr {
	template := `
	package abc

	var _ = {{TEXT}}
	`

	packageText := strings.Replace(template, "{{TEXT}}", input, 1)

	fset := token.NewFileSet()
	tree, err := parser.ParseFile(fset, "", packageText, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	var found ast.Expr

	ast.Inspect(tree, func(node ast.Node) bool {
		switch typedNode := node.(type) {
		case ast.Expr:
			found = typedNode
			return false
		default:
			return true
		}
	})
	if found == nil {
		panic("expr not found")
	}

	return found
}
