package filegetter

import (
	"errors"
	"go/ast"
	"go/token"

	"component-generator/internal/naming"
)

type Implementator struct {
	err         error
	packageName string
}

func New(packageName string) *Implementator {
	return &Implementator{
		packageName: packageName,
	}
}

func (i *Implementator) Name() string {
	return "filegetter"
}

func (i *Implementator) Description() string {
	return "Generates a function that reads a file and unmarshals it"
}

func (i *Implementator) Error() error {
	return i.err
}

func (i *Implementator) Visit(node ast.Node) (bool, []ast.Decl) {
	decls := []ast.Decl{}

	switch n := node.(type) {
	case *ast.FuncType:
		err := validateReturnList(n.Results.List)
		if err != nil {
			i.err = err
			return false, nil
		}

		generated, err := tree(n, i.packageName)
		if err != nil {
			i.err = err
			return false, nil
		}

		decls = append(decls, generated)

	case *ast.InterfaceType:
		i.err = errors.New("filegetter doesnt work on interfaces")
		return false, nil
	default:
		return true, nil
	}

	return false, decls
}

func tree(fn *ast.FuncType, packageName string) (*ast.FuncDecl, error) {
	returnList := possiblyAddPackageName(fn.Results.List, packageName)

	returnType := returnList[0].Type

	resultVarName := naming.VariableNameFromExpr(returnType)

	varIdent := ast.NewIdent(resultVarName)

	var zeroValue ast.Expr

	var unmarshalArg ast.Expr

	switch t := returnType.(type) {
	case *ast.StarExpr:
		zeroValue = ast.NewIdent("nil")
		unmarshalArg = varIdent
	case *ast.SelectorExpr:
		zeroValue = &ast.CompositeLit{
			Type: t,
		}

		unmarshalArg = &ast.UnaryExpr{
			Op: token.AND,
			X:  varIdent,
		}
	default:
		return nil, errors.New("unsupported return type")
	}

	ignoredField := []*ast.Field{}

	for _, f := range fn.Params.List {
		idents := []*ast.Ident{}

		if len(f.Names) != 0 {
			for range f.Names {
				idents = append(idents, ast.NewIdent("_"))
			}
		} else {
			idents = append(idents, ast.NewIdent("_"))
		}

		ignoredField = append(ignoredField, &ast.Field{
			Names: idents,
			Type:  f.Type,
		})
	}

	var firstError []ast.Stmt
	var secondError []ast.Stmt
	if len(returnList) == 1 {
		firstError = append(firstError, &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: ast.NewIdent("panic"),
				Args: []ast.Expr{
					ast.NewIdent(
						`"failed to read file"`,
					),
				},
			},
		})
		secondError = append(secondError, &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: ast.NewIdent("panic"),
				Args: []ast.Expr{
					ast.NewIdent(
						`"failed to unmarshal json"`,
					),
				},
			},
		})
	}
	if len(returnList) == 2 {
		firstError = []ast.Stmt{
			&ast.ReturnStmt{
				Results: []ast.Expr{
					zeroValue,
					&ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   ast.NewIdent("fmt"),
							Sel: ast.NewIdent("Errorf"),
						},
						Args: []ast.Expr{
							ast.NewIdent(
								`"failed to read %s: %w"`,
							),
							ast.NewIdent("path"),
							ast.NewIdent("err"),
						},
					},
				},
			},
		}
		secondError = []ast.Stmt{
			&ast.ReturnStmt{
				Results: []ast.Expr{
					zeroValue,
					&ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   ast.NewIdent("fmt"),
							Sel: ast.NewIdent("Errorf"),
						},
						Args: []ast.Expr{
							ast.NewIdent(
								`"failed to unmarshal json: %w"`,
							),
							ast.NewIdent("err"),
						},
					},
				},
			},
		}
	}

	return &ast.FuncDecl{
		Name: ast.NewIdent("StateGetter"),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("path")},
						Type:  ast.NewIdent("string"),
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: &ast.FuncType{
							Params: fn.Params,
							Results: &ast.FieldList{
								List: returnList,
							},
						},
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.FuncLit{
							Type: &ast.FuncType{
								Params: &ast.FieldList{
									List: ignoredField,
								},
								Results: &ast.FieldList{
									List: returnList,
								},
							},
							Body: &ast.BlockStmt{
								List: []ast.Stmt{
									// bytes, err := os.ReadFile(path)
									&ast.AssignStmt{
										Lhs: []ast.Expr{
											ast.NewIdent("bytes"),
											ast.NewIdent("err"),
										},
										Tok: token.DEFINE,
										Rhs: []ast.Expr{
											&ast.CallExpr{
												Fun: &ast.SelectorExpr{
													X:   ast.NewIdent("os"),
													Sel: ast.NewIdent("ReadFile"),
												},
												Args: []ast.Expr{
													ast.NewIdent("path"),
												},
											},
										},
									},
									// if err != nil { ... }
									&ast.IfStmt{
										Cond: &ast.BinaryExpr{
											X:  ast.NewIdent("err"),
											Op: token.NEQ,
											Y:  ast.NewIdent("nil"),
										},
										Body: &ast.BlockStmt{
											List: firstError,
										},
									},
									// var {{varIdent}} {{returnType}}
									&ast.DeclStmt{
										Decl: &ast.GenDecl{
											Tok: token.VAR,
											Specs: []ast.Spec{
												&ast.ValueSpec{
													Names: []*ast.Ident{varIdent},
													Type:  returnType,
												},
											},
										},
									},
									// err = json.Unmarshal(bytes, {{unmarshalArg}})
									&ast.AssignStmt{
										Lhs: []ast.Expr{
											ast.NewIdent("err"),
										},
										Tok: token.ASSIGN,
										Rhs: []ast.Expr{
											&ast.CallExpr{
												Fun: &ast.SelectorExpr{
													X:   ast.NewIdent("json"),
													Sel: ast.NewIdent("Unmarshal"),
												},
												Args: []ast.Expr{
													ast.NewIdent("bytes"),
													unmarshalArg,
												},
											},
										},
									},
									// if err != nil { ... }
									&ast.IfStmt{
										Cond: &ast.BinaryExpr{
											X:  ast.NewIdent("err"),
											Op: token.NEQ,
											Y:  ast.NewIdent("nil"),
										},
										Body: &ast.BlockStmt{
											List: secondError,
										},
									},
									// return {{varIdent}}, nil
									&ast.ReturnStmt{
										Results: []ast.Expr{
											varIdent,
											ast.NewIdent("nil"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}, nil
}

func validateReturnList(returnList []*ast.Field) error {
	if len(returnList) == 0 || len(returnList) > 2 {
		return errors.New("return list must have 1 or 2 elements")
	}

	if len(returnList) == 1 {
		return nil
	}

	secondReturn := returnList[1]

	switch t := secondReturn.Type.(type) {
	case *ast.Ident:
		if t.Name != "error" {
			return errors.New("second return value must be of type error")
		}
	}

	return nil
}

func possiblyAddPackageName(fields []*ast.Field, packageName string) []*ast.Field {
	var newType ast.Expr
	switch t := fields[0].Type.(type) {
	case *ast.Ident:
		// add package name to the type
		newType = &ast.SelectorExpr{
			X:   ast.NewIdent(packageName),
			Sel: t,
		}
	case *ast.StarExpr:
		switch x := t.X.(type) {
		case *ast.Ident:
			// add package name to the type
			newType = &ast.SelectorExpr{
				X:   ast.NewIdent(packageName),
				Sel: x,
			}
		}

		newType = &ast.StarExpr{
			X: newType,
		}
	}

	fields[0].Type = newType

	return fields
}
