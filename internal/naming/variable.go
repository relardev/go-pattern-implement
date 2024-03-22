package naming

import (
	"fmt"
	"go/ast"
	"strings"
	"unicode/utf8"
)

func VariableNameFromExpr(t ast.Expr) string {
	switch r := t.(type) {
	case *ast.StarExpr:
		return VariableNameFromExpr(r.X)
	case *ast.SelectorExpr:
		return nameFromSelector(r)
	case *ast.Ident:
		return LowercaseFirstLetter(r.Name)
	case *ast.ArrayType:
		return VariableNameFromExpr(r.Elt) + "s"
	case *ast.MapType:
		return VariableNameFromExpr(r.Value) + "s"
	default:
		panic(fmt.Sprintf("Unknown type in VariableNameFromExpr: %T", r))
	}
}

func LowercaseFirstLetter(s string) string {
	if s == "" {
		return ""
	}
	// Get the first rune
	r, size := utf8.DecodeRuneInString(s)
	// Lowercase the first rune and concatenate with the rest of the string
	return strings.ToLower(string(r)) + s[size:]
}

func nameFromSelector(sel *ast.SelectorExpr) string {
	return LowercaseFirstLetter(sel.Sel.Name)
}
