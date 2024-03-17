package cache

import (
	"component-generator/internal/code"
	"fmt"
	"go/ast"
	"go/token"
	"unicode"
)

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
	return "cache"
}

func (i *Implementator) Description() string {
	return "Cache results of wrapped interface"
}

func (i *Implementator) Error() error {
	return i.err
}

func (i *Implementator) Visit(node ast.Node) (bool, []ast.Decl) {
	decls := []ast.Decl{}

	switch typeSpec := node.(type) {
	case *ast.TypeSpec:
		decls = append(decls, code.Struct(
			"Cache",
			code.FieldFromTypeSpec(typeSpec, i.packageName),
			code.StructField{
				Name:    "cache",
				TypeStr: "*cache.Cache",
			},
		))
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
					{
						Names: []*ast.Ident{
							ast.NewIdent("expiration"),
							ast.NewIdent("cleanupInterval"),
						},
						Type: ast.NewIdent("time.Duration"),
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
									&ast.KeyValueExpr{
										Key: ast.NewIdent("cache"),
										Value: &ast.CallExpr{
											Fun: ast.NewIdent("cache.New"),
											Args: []ast.Expr{
												ast.NewIdent("expiration"),
												ast.NewIdent("cleanupInterval"),
											},
										},
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
		Results: field.Type.(*ast.FuncType).Results,
	}

	return &ast.FuncDecl{
		Name: ast.NewIdent(funcName),
		Type: typeDef,
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
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						ast.NewIdent("key"),
					},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						ast.NewIdent("\"\""),
					},
				},
				// cachedItem, found := r.cache.Get(cacheKey)
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						ast.NewIdent("cachedItem"),
						ast.NewIdent("found"),
					},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X: &ast.SelectorExpr{
									X:   ast.NewIdent(firstLetter),
									Sel: ast.NewIdent("cache"),
								},
								Sel: ast.NewIdent("Get"),
							},
							Args: []ast.Expr{
								ast.NewIdent("key"),
							},
						},
					},
				},
				// if found { ... }
				&ast.IfStmt{
					Cond: ast.NewIdent("found"),
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							// recipes, ok := cachedItem.([]scenario.Recipe)
							&ast.AssignStmt{
								Lhs: []ast.Expr{
									ast.NewIdent("recipes"),
									ast.NewIdent("ok"),
								},
								Tok: token.DEFINE,
								Rhs: []ast.Expr{
									&ast.TypeAssertExpr{
										X: ast.NewIdent("cachedItem"),
										Type: &ast.ArrayType{
											Elt: &ast.SelectorExpr{
												X:   ast.NewIdent("scenario"),
												Sel: ast.NewIdent("Recipe"),
											},
										},
									},
								},
							},
							// if !ok { return nil, errors.New("invalid object in cache") }
							&ast.IfStmt{
								Cond: &ast.UnaryExpr{
									Op: token.NOT,
									X:  ast.NewIdent("ok"),
								},
								Body: &ast.BlockStmt{
									List: []ast.Stmt{
										&ast.ReturnStmt{
											Results: []ast.Expr{
												ast.NewIdent("nil"),
												&ast.CallExpr{
													Fun: &ast.SelectorExpr{
														X:   ast.NewIdent("errors"),
														Sel: ast.NewIdent("New"),
													},
													Args: []ast.Expr{
														&ast.BasicLit{
															Kind:  token.STRING,
															Value: `"invalid object in cache"`,
														},
													},
												},
											},
										},
									},
								},
							},
							// return recipes, nil
							&ast.ReturnStmt{
								Results: []ast.Expr{
									ast.NewIdent("recipes"),
									ast.NewIdent("nil"),
								},
							},
						},
					},
				},

				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent(string(firstLetter)),
								Sel: ast.NewIdent(funcName),
							},
							Args: []ast.Expr{
								&ast.SelectorExpr{
									X:   ast.NewIdent("cache"),
									Sel: ast.NewIdent("Get"),
								},
							},
						},
					},
				},
			},
		},
	}
}
