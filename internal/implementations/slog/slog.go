package slog

import (
	"go/ast"
	"go/token"
)

func Visitor(packageName string, fset *token.FileSet) func(node ast.Node) bool {
	return func(node ast.Node) bool {
		return true
	}
}
