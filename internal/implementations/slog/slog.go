package slog

import (
	"component-generator/internal/code"
	"component-generator/internal/naming"
	"fmt"
	"go/ast"
	"go/token"
	"unicode"
)

type slogFunc string

const (
	slogInfo slogFunc = "Info"
)

func Visitor(packageName string, fset *token.FileSet) func(node ast.Node) bool {
	return func(node ast.Node) bool {
		return true
	}
}

type Implementator struct {
	err         error
	packageName string
}

func New(sourcePackageName string) *Implementator {
	return &Implementator{
		packageName: sourcePackageName,
	}
}

func (i *Implementator) Name() string {
	return "slog"
}

func (i *Implementator) Description() string {
	return "Generates slog stdout for a given interface, expect only single return errors in methods."
}

func (i *Implementator) Error() error {
	return i.err
}

func (i *Implementator) Visit(node ast.Node) (bool, []ast.Decl) {
	decls := []ast.Decl{}

	switch typeSpec := node.(type) {
	case *ast.TypeSpec:
		decls = append(decls, code.Struct(typeSpec.Name.Name))
		decls = append(decls, newFunction(typeSpec.Name.Name))

		switch interfaceNode := typeSpec.Type.(type) {
		case *ast.InterfaceType:
			for _, methodDef := range interfaceNode.Methods.List {
				decls = append(decls, i.implementFunction(typeSpec.Name.Name, methodDef))
			}
		default:
			panic("not an interface")
		}
	default:
		return true, nil
	}
	return false, decls
}

func newFunction(interfaceName string) ast.Decl {
	return &ast.FuncDecl{
		Name: ast.NewIdent("New" + interfaceName),
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: &ast.StarExpr{
							X: ast.NewIdent(interfaceName),
						},
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
							},
						},
					},
				},
			},
		},
	}
}

func (i *Implementator) implementFunction(interfaceName string, field *ast.Field) ast.Decl {
	firstLetter := string(unicode.ToLower(rune(interfaceName[0])))
	funcName := field.Names[0].Name

	typeDef := &ast.FuncType{
		Params: &ast.FieldList{
			List: field.Type.(*ast.FuncType).Params.List,
		},
		Results: &ast.FieldList{},
	}

	callArgs := naming.ExtractFuncArgs(field)

	if field.Type != nil {
		switch n := field.Type.(type) {
		case *ast.FuncType:
			if n.Results != nil {
				typeDef.Results.List = append(typeDef.Results.List, n.Results.List...)
			}
		default:
			fmt.Printf("Unknown type: %T\n", n)
			panic("Unknown type")
		}
	}

	returns := processReturns(typeDef)

	interfacePlusMethodName := fmt.Sprintf(
		"%s_%s",
		naming.LowercaseFirstLetter(interfaceName),
		naming.LowercaseFirstLetter(funcName),
	)

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
		Body: i.stdoutBody(returns, interfacePlusMethodName, callArgs),
	}
}

func (i *Implementator) stdoutBody(
	returnExpr []ast.Expr,
	logPrefix string,
	funcArgs []ast.Expr,
) *ast.BlockStmt {
	slogArgs := []ast.Expr{
		&ast.BasicLit{
			Kind:  token.STRING,
			Value: fmt.Sprintf(`%q`, fmt.Sprintf("%s: ", logPrefix)),
		},
	}

	for _, arg := range funcArgs {
		slogArgs = append(slogArgs, &ast.BasicLit{
			Kind:  token.STRING,
			Value: fmt.Sprintf(`%q`, arg),
		})
		slogArgs = append(slogArgs, &ast.BasicLit{
			Kind:  token.STRING,
			Value: fmt.Sprintf("%s", arg),
		})
	}

	blockStmt := &ast.BlockStmt{
		List: []ast.Stmt{
			&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   ast.NewIdent("slog"),
						Sel: ast.NewIdent(string(slogInfo)),
					},
					Args: slogArgs,
				},
			},
		},
	}

	if len(returnExpr) > 0 {
		blockStmt.List = append(blockStmt.List, &ast.ReturnStmt{
			Results: returnExpr,
		})
	}

	return blockStmt
}

func processReturns(typeDef *ast.FuncType) []ast.Expr {
	if len(typeDef.Results.List) == 0 {
		return []ast.Expr{}
	}

	if len(typeDef.Results.List) > 1 {
		panic("Slog implementation only supports returning single error values")
	}

	result := typeDef.Results.List[0]
	switch n := result.Type.(type) {
	case *ast.Ident:
		if n.Name == "error" {
			return []ast.Expr{ast.NewIdent("nil")}
		}
	default:
		panic("Slog implementation only supports returning error")
	}
	return []ast.Expr{}
}
