package naming

import (
	"fmt"
	"go/ast"
	"strings"
	"unicode"
	"unicode/utf8"
)

func ExtractFuncArgs(field *ast.Field) []ast.Expr {
	callArgs := []ast.Expr{}
	usedNames := map[string]int{}
	for _, param := range field.Type.(*ast.FuncType).Params.List {
		var name string
		switch n := param.Type.(type) {
		case *ast.Ident:
			name = "arg"
		case *ast.StarExpr:
			name = "arg"
		case *ast.SelectorExpr:
			name = VariableNameFromExpr(n)
		case *ast.ArrayType:
			name = VariableNameFromExpr(n)
		case *ast.MapType:
			name = VariableNameFromExpr(n)
		case *ast.FuncType:
			name = "fn"
		default:
			name = "arg"
		}

		if _, ok := usedNames[name]; ok {
			usedNames[name]++
			name = fmt.Sprintf("%s%d", name, usedNames[name])
		} else {
			usedNames[name] = 1
		}

		if len(param.Names) == 0 {
			param.Names = []*ast.Ident{ast.NewIdent(name)}
			callArgs = append(callArgs, ast.NewIdent(name))
		} else {
			for _, name := range param.Names {
				callArgs = append(callArgs, ast.NewIdent(name.Name))
			}
		}
	}

	return callArgs
}

func ExtractFuncReturns(field *ast.Field) []ast.Expr {
	returns := []ast.Expr{}
	results := field.Type.(*ast.FuncType).Results
	if results == nil {
		return returns
	}

	for _, result := range results.List {
		returns = append(returns, ast.NewIdent(VariableNameFromExpr(result.Type)))
	}

	return returns
}

func VariableNameFromExpr(t ast.Expr) string {
	switch r := t.(type) {
	case *ast.StarExpr:
		return VariableNameFromExpr(r.X)
	case *ast.SelectorExpr:
		return nameFromSelector(r)
	case *ast.Ident:
		if unicode.IsUpper(rune(r.Name[0])) {
			return VarNameFromType(r.Name)
		}
		switch r.Name {
		case "error":
			return "err"
		case "string":
			return "str"
		case "int":
			return "i"
		case "bool":
			return "b"
		case "uint64":
			return "u64"
		default:
			panic(fmt.Sprintf("Unhandled ident in VariableNameFromExpr: %s", r.Name))
		}
	case *ast.ArrayType:
		name := VariableNameFromExpr(r.Elt)
		if !strings.HasSuffix(name, "s") {
			name += "s"
		}
		return name
	case *ast.MapType:
		name := VariableNameFromExpr(r.Value)
		if !strings.HasSuffix(name, "s") {
			name += "s"
		}
		return name

	case *ast.InterfaceType:
		return "thing"
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

func VarNameFromType(s string) string {
	return LowercaseFirstLetter(s)
	// TODO lowercase everything untill upper followed by lower found
	// to handle case like CDPAttribute
	// if len(s) == 1 {
	// 	return LowercaseFirstLetter(s)
	// }
	// result := ""
	// for i, c := range s {
	// 	if s[i+1] > len(s) {
	// 		break
	// 	}
	// 	if s[i+1]
	// }
}

func nameFromSelector(sel *ast.SelectorExpr) string {
	if sel.Sel.Name == "Context" {
		return "ctx"
	}
	return LowercaseFirstLetter(sel.Sel.Name)
}
