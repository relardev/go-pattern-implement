package code

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
)

func NodeToString(node any) string {
	switch n := node.(type) {
	case *ast.FieldList:
		return printFuncType(n)
	case *ast.Field:
		return printField(n)
	case []ast.Expr:
		return printExprs(n)
	case ast.Expr, ast.Stmt, ast.Decl, ast.Spec, []ast.Stmt, []ast.Decl, []ast.Spec:
		fset := token.NewFileSet()

		var output strings.Builder
		if err := printer.Fprint(&output, fset, node); err != nil {
			panic(err)
		}

		return output.String()
	default:
		panic(fmt.Sprintf("unsupported node type in NodeToString: %T", node))
	}
}

func printFuncType(fl *ast.FieldList) string {
	var params []string

	if fl == nil {
		return ""
	}

	for _, p := range fl.List {
		params = append(params, printField(p))
	}

	return join(params, ", ")
}

func printField(f *ast.Field) string {
	if f.Names == nil {
		return exprString(f.Type)
	}

	var names []string
	for _, n := range f.Names {
		names = append(names, n.Name)
	}

	return join(names, ", ") + " " + exprString(f.Type)
}

func printExprs(exprs []ast.Expr) string {
	var exprStrs []string
	for _, e := range exprs {
		exprStrs = append(exprStrs, exprString(e))
	}

	return join(exprStrs, ", ")
}

func exprString(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return exprString(x.X) + "." + x.Sel.Name
	case *ast.StarExpr:
		return "*" + exprString(x.X)
	case *ast.ArrayType:
		return "[]" + exprString(x.Elt)
	case *ast.MapType:
		return "map[" + exprString(x.Key) + "]" + exprString(x.Value)
	case *ast.CompositeLit:
		return fmt.Sprintf("%s{%s}", exprString(x.Type), printExprs(x.Elts))
	case *ast.CallExpr:
		return fmt.Sprintf("%s(%s)", exprString(x.Fun), printExprs(x.Args))
	case *ast.BasicLit:
		return x.Value
	case *ast.IndexExpr:
		return fmt.Sprintf("%s[%s]", exprString(x.X), exprString(x.Index))
	case *ast.InterfaceType:
		return "interface{}"
	default:
		// this might blow up? before it was the return below but not sure
		// if it was important
		// return fmt.Sprintf("%T", e)
		panic(fmt.Sprintf("unsupported type in exprString: %T", e))
	}
}

func join(strs []string, sep string) string {
	var buf bytes.Buffer
	for i, s := range strs {
		if i > 0 {
			buf.WriteString(sep)
		}
		buf.WriteString(s)
	}
	return buf.String()
}
