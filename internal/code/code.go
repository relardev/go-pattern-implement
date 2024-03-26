package code

import (
	"component-generator/internal/naming"
	"fmt"
	"go/ast"
	"go/token"
	"unicode"
)

type StructField struct {
	Name string

	// We need either TypeStr or TypeSpec
	TypeStr  string
	TypeSpec ast.Expr
}

func Struct(name string, fields ...StructField) ast.Decl {
	specs := []*ast.Field{}
	for _, field := range fields {
		var t ast.Expr
		if field.TypeSpec != nil {
			t = field.TypeSpec
		} else {
			t = ast.NewIdent(field.TypeStr)
		}

		specs = append(specs, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(field.Name)},
			Type:  t,
		})
	}
	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent(name),
				Type: &ast.StructType{
					Fields: &ast.FieldList{
						List: specs,
					},
				},
			},
		},
	}
}

func FieldFromTypeSpec(typeSpec *ast.TypeSpec, packageName string) StructField {
	name := typeSpec.Name.Name
	lowerFirstLetter := unicode.ToLower(rune(name[0]))
	return StructField{
		Name:    string(lowerFirstLetter),
		TypeStr: packageName + "." + name,
	}
}

func IfErrReturnErr(additionalReturns ...ast.Expr) *ast.IfStmt {
	additionalReturns = append(additionalReturns, ast.NewIdent("err"))
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  ast.NewIdent("err"),
			Op: token.NEQ,
			Y:  ast.NewIdent("nil"),
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: additionalReturns,
				},
			},
		},
	}
}

func IsContext(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.SelectorExpr:
		x, ok := t.X.(*ast.Ident)
		return ok && x.Name == "context" && t.Sel.Name == "Context"
	default:
		return false
	}
}

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
			name = naming.VariableNameFromExpr(n)
		case *ast.ArrayType:
			name = "arg"
		case *ast.MapType:
			name = "arg"
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

func AddPackageNameToResults(results *ast.FieldList, packageName string) *ast.FieldList {
	if results == nil {
		return nil
	}

	for _, r := range results.List {
		r.Type = PossiblyAddPackageName(packageName, r.Type)
	}

	return results
}

func PossiblyAddPackageName(packageName string, expr ast.Expr) ast.Expr {
	var newExpr ast.Expr
	switch t := expr.(type) {
	case *ast.Ident:
		isFirstLetterUpper := unicode.IsUpper(rune(t.Name[0]))
		if isFirstLetterUpper {
			newExpr = &ast.SelectorExpr{
				X:   ast.NewIdent(packageName),
				Sel: t,
			}
		} else {
			newExpr = t
		}
	case *ast.StarExpr:
		newExpr = &ast.StarExpr{
			X: PossiblyAddPackageName(packageName, t.X),
		}
	case *ast.ArrayType:
		newExpr = &ast.ArrayType{
			Elt: PossiblyAddPackageName(packageName, t.Elt),
		}

	case *ast.SelectorExpr:
		return t

	case *ast.MapType:
		newExpr = &ast.MapType{
			Key:   PossiblyAddPackageName(packageName, t.Key),
			Value: PossiblyAddPackageName(packageName, t.Value),
		}
	default:
		panic(fmt.Sprintf("unsupported type in PossiblyAddPackageName: %T", t))
	}

	return newExpr
}

func ZeroValue(t ast.Expr) ast.Expr {
	switch t := t.(type) {
	case *ast.StarExpr, *ast.ArrayType, *ast.MapType:
		return ast.NewIdent("nil")
	case *ast.SelectorExpr:
		return &ast.CompositeLit{
			Type: t,
		}
	case *ast.Ident:
		switch t.Name {
		case "error":
			return ast.NewIdent("nil")
		case "string":
			return &ast.BasicLit{
				Kind:  token.STRING,
				Value: "\"\"",
			}
		case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
			return &ast.BasicLit{
				Kind:  token.INT,
				Value: "0",
			}
		case "float32", "float64":
			return &ast.BasicLit{
				Kind:  token.FLOAT,
				Value: "0.0",
			}
		case "bool":
			return ast.NewIdent("false")
		default:
			return &ast.CompositeLit{
				Type: t,
			}
		}
	default:
		panic(fmt.Sprintf("unsupported type in Zero Value: %T", t))
	}
}

func IsError(t ast.Expr) bool {
	switch t := t.(type) {
	case *ast.Ident:
		return t.Name == "error"
	default:
		return false
	}
}

// DoesFieldReturnError returns true if the last return value of the function is an error
func DoesFieldReturnError(field *ast.Field) (bool, int) {
	results := field.Type.(*ast.FuncType).Results
	return DoesFieldListReturnError(results)
}

// DoesFieldListtReturnError returns true if the last field in a FieldList is an error
func DoesFieldListReturnError(results *ast.FieldList) (bool, int) {
	if results == nil {
		return false, 0
	}
	lastPosition := len(results.List) - 1
	return IsError(results.List[lastPosition].Type), lastPosition
}

func RemoveNames(fl *ast.FieldList) *ast.FieldList {
	flCopy := &ast.FieldList{
		List: make([]*ast.Field, len(fl.List)),
	}
	for i, f := range fl.List {
		flCopy.List[i] = &ast.Field{
			Type: f.Type,
		}
	}
	return flCopy
}
