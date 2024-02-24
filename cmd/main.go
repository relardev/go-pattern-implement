package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"unicode"
)

const mainTemplate = `
package xxx

type Something interface {
	DoSomething(string) error
}

`

func main() {
	// args
	packageName := "xxx"

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "main.go", mainTemplate, 0)
	if err != nil {
		panic(err)
	}

	visitor := func(node ast.Node) bool {
		if node == nil {
			return false
		}

		decls := []ast.Decl{}

		switch typeSpec := node.(type) {
		case *ast.TypeSpec:
			decls = append(decls, structFromInterface(typeSpec.Name.Name, packageName))
			decls = append(decls, newWraperFunction(typeSpec.Name.Name, packageName))

			switch interfaceNode := typeSpec.Type.(type) {
			case *ast.InterfaceType:
				for _, methodDef := range interfaceNode.Methods.List {
					decls = append(decls, implementFunction(typeSpec.Name.Name, methodDef))
				}
			default:
				panic("not an interface")
			}
		default:
			return true
		}

		printer.Fprint(os.Stdout, fset, decls)

		return false
	}

	ast.Inspect(f, visitor)
}

func structFromInterface(interfaceName, interfacePackage string) ast.Decl {
	firstLetter := unicode.ToLower(rune(interfaceName[0]))
	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent(interfaceName),
				Type: &ast.StructType{
					Fields: &ast.FieldList{
						List: []*ast.Field{
							{
								Names: []*ast.Ident{ast.NewIdent(string(firstLetter))},
								Type: ast.NewIdent(
									fmt.Sprintf("%s.%s", interfacePackage, interfaceName),
								),
							},
						},
					},
				},
			},
		},
	}
}

func newWraperFunction(interfaceName, interfacePackage string) ast.Decl {
	firstLetter := unicode.ToLower(rune(interfaceName[0]))
	return &ast.FuncDecl{
		Name: ast.NewIdent("New" + interfaceName),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent(string(firstLetter))},
						Type: ast.NewIdent(
							fmt.Sprintf("%s.%s", interfacePackage, interfaceName),
						),
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: ast.NewIdent(
							interfaceName,
						),
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.UnaryExpr{
							Op: token.AND,
							X: &ast.CompositeLit{
								Type: ast.NewIdent(interfaceName),
								Elts: []ast.Expr{
									&ast.KeyValueExpr{
										Key:   ast.NewIdent(string(firstLetter)),
										Value: ast.NewIdent(string(firstLetter)),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func implementFunction(interfaceName string, field *ast.Field) ast.Decl {
	firstLetter := string(unicode.ToLower(rune(interfaceName[0])))
	funcName := field.Names[0].Name

	typeDef := &ast.FuncType{
		Params:  &ast.FieldList{},
		Results: &ast.FieldList{},
	}

	callArgs := []ast.Expr{}

	for i, param := range field.Type.(*ast.FuncType).Params.List {
		if len(param.Names) == 0 {
			param.Names = []*ast.Ident{ast.NewIdent("arg" + fmt.Sprint(i))}
		}

		callArgs = append(callArgs, ast.NewIdent(param.Names[0].Name))

		typeDef.Params.List = append(typeDef.Params.List, param)
	}

	typeDef.Results.List = append(typeDef.Results.List, field.Type.(*ast.FuncType).Results.List...)

	return &ast.FuncDecl{
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent(firstLetter)},
					Type: &ast.StarExpr{
						X: ast.NewIdent(interfaceName),
					},
				},
			},
		},
		Name: ast.NewIdent(funcName),
		Type: typeDef,
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent(fmt.Sprintf("%s.%s", firstLetter, firstLetter)),
								Sel: ast.NewIdent(funcName),
							},
							Args: callArgs,
						},
					},
				},
			},
		},
	}
}
