package metrics

import (
	"component-generator/internal/code"
	"fmt"
	"go/ast"
	"go/token"
	"unicode"

	naming "component-generator/internal/naming"
)

type Implementator struct {
	err                      error
	packageName              string
	implementatorName        string
	observabilityPackageName string
}

func New(sourcePackageName, implementatorName, observabilityPackageName string) *Implementator {
	return &Implementator{
		packageName:              sourcePackageName,
		implementatorName:        implementatorName,
		observabilityPackageName: observabilityPackageName,
	}
}

func (i *Implementator) Name() string {
	return i.implementatorName
}

func (i *Implementator) Description() string {
	return "Generates observability metrics for a given interface"
}

func (i *Implementator) Error() error {
	return i.err
}

func (i *Implementator) Visit(node ast.Node) (bool, []ast.Decl) {
	decls := []ast.Decl{}

	switch typeSpec := node.(type) {
	case *ast.TypeSpec:
		decls = append(decls, code.Struct(typeSpec.Name.Name, code.FieldFromTypeSpec(typeSpec, i.packageName)))
		decls = append(decls, newWraperFunction(typeSpec.Name.Name, i.packageName))

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

func (i *Implementator) implementFunction(interfaceName string, field *ast.Field) ast.Decl {
	firstLetter := string(unicode.ToLower(rune(interfaceName[0])))
	funcName := field.Names[0].Name

	typeDef := &ast.FuncType{
		Params: &ast.FieldList{
			List: field.Type.(*ast.FuncType).Params.List,
		},
		Results: &ast.FieldList{},
	}

	callArgs := code.ExtractFuncArgs(
		field,
	)

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

	returns, returningError := processReturns(typeDef)

	callWrapped := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent(fmt.Sprintf("%s.%s", firstLetter, firstLetter)),
			Sel: ast.NewIdent(funcName),
		},
		Args: callArgs,
	}

	var callStmt ast.Stmt

	var returnExpr []ast.Expr

	if returningError {
		returnExpr = returns

		callStmt = &ast.AssignStmt{
			Lhs: returnExpr,
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				callWrapped,
			},
		}
	} else {
		returnExpr = []ast.Expr{
			callWrapped,
		}
	}

	measurePrefix := fmt.Sprintf(
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
		Body: i.measuredBody(callStmt, returnExpr, measurePrefix),
	}
}

func (i *Implementator) measuredBody(
	callStmt ast.Stmt,
	returnExpr []ast.Expr,
	measurePrefix string,
) *ast.BlockStmt {
	blockStmt := &ast.BlockStmt{
		List: []ast.Stmt{
			&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   ast.NewIdent(i.observabilityPackageName),
						Sel: ast.NewIdent("Increment"),
					},
					Args: []ast.Expr{
						&ast.BasicLit{
							Kind:  token.STRING,
							Value: fmt.Sprintf(`%q`, measurePrefix),
						},
					},
				},
			},
			&ast.DeferStmt{
				Call: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   ast.NewIdent(i.observabilityPackageName),
						Sel: ast.NewIdent("ObserveDuration"),
					},
					Args: []ast.Expr{
						&ast.BasicLit{
							Kind:  token.STRING,
							Value: fmt.Sprintf(`"%s_seconds"`, measurePrefix),
						},
						&ast.CallExpr{
							Fun: ast.NewIdent("time.Now"),
						},
					},
				},
			},
		},
	}

	reportErr := &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  ast.NewIdent("err"),
			Op: token.NEQ,
			Y:  ast.NewIdent("nil"),
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   ast.NewIdent(i.observabilityPackageName),
							Sel: ast.NewIdent("Increment"),
						},
						Args: []ast.Expr{
							&ast.BasicLit{
								Kind:  token.STRING,
								Value: fmt.Sprintf(`%q`, measurePrefix+"_error"),
							},
						},
					},
				},
			},
		},
	}

	returnStmt := &ast.ReturnStmt{
		Results: returnExpr,
	}

	if callStmt != nil {
		blockStmt.List = append(blockStmt.List, callStmt)
		blockStmt.List = append(blockStmt.List, reportErr)
	}

	blockStmt.List = append(blockStmt.List, returnStmt)

	return blockStmt
}

func processReturns(typeDef *ast.FuncType) ([]ast.Expr, bool) {
	resultsList := typeDef.Results.List
	var returningError bool

	var returns []ast.Expr
	for _, result := range resultsList {
		switch n := result.Type.(type) {
		case *ast.Ident:
			if n.Name == "error" {
				returningError = true
			}
		}
	}

	var namedReturns int

	if returningError {
		namedReturns = len(resultsList) - 1
	} else {
		namedReturns = len(resultsList)
	}

	for n, result := range resultsList {
		isError := false

		switch n := result.Type.(type) {
		case *ast.Ident:
			if n.Name == "error" {
				isError = true
			}
		}

		if isError {
			returns = append(returns, ast.NewIdent("err"))
		} else {
			if namedReturns > 1 {
				returns = append(returns, ast.NewIdent(fmt.Sprintf("result%v", n+1)))
			} else {
				returns = append(returns, ast.NewIdent("result"))
			}
		}
	}

	return returns, returningError
}
