package store

import (
	"go/ast"
	"go/token"

	"component-generator/internal/code"
	"component-generator/internal/naming"
)

type Implementator struct {
	err           error
	packageName   string
	interfaceName string
	methodDef     *ast.Field
	funcName      string
	variableName  string
	lockName      string
}

func New(sourcePackageName string) *Implementator {
	return &Implementator{
		packageName: sourcePackageName,
	}
}

func (i *Implementator) Name() string {
	return "store"
}

func (i *Implementator) Error() error {
	return i.err
}

func (i *Implementator) Visit(node ast.Node) (bool, []ast.Decl) {
	decls := []ast.Decl{}

	switch typeSpec := node.(type) {
	case *ast.TypeSpec:
		switch interfaceNode := typeSpec.Type.(type) {
		case *ast.InterfaceType:
			if len(interfaceNode.Methods.List) != 1 {
				panic("interface should have only one method")
			}
			i.methodDef = interfaceNode.Methods.List[0]

			returns := i.methodDef.Type.(*ast.FuncType).Results.List

			if len(returns) != 2 {
				panic("method should have two returns")
			}

			if len(i.methodDef.Type.(*ast.FuncType).Params.List) != 0 {
				panic("method should have no parameters")
			}

			i.interfaceName = typeSpec.Name.Name
			i.variableName = naming.VariableNameFromExpr(returns[0].Type)
			i.lockName = i.variableName + "Lock"
			resultType := returns[0].Type

			field := code.FieldFromTypeSpec(typeSpec, i.packageName)

			fields := []code.StructField{
				{
					Name:    "repo",
					TypeStr: field.TypeStr,
				},
				{
					Name:    "interval",
					TypeStr: "time.Duration",
				},
				{
					Name:    i.lockName,
					TypeStr: "sync.RWMutex",
				},
				{
					Name:     i.variableName,
					TypeSpec: resultType,
				},
			}
			decls = append(decls, code.Struct("Store", fields...))

			i.funcName = i.methodDef.Names[0].Name
			decls = append(decls, newFunc(field.TypeStr, i.packageName))
			decls = append(decls, i.loopFunc())
			decls = append(decls, i.loadFunc())
			decls = append(decls, i.implementFunction())
		default:
			panic("not an interface")
		}
	default:
		return true, nil
	}
	return false, decls
}

func newFunc(interfaceName, interfacePackage string) ast.Decl {
	newFunc := &ast.FuncDecl{
		Name: ast.NewIdent("New"),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("repo")},
						Type:  ast.NewIdent(interfaceName),
					},
					{
						Names: []*ast.Ident{ast.NewIdent("interval")},
						Type:  ast.NewIdent("time.Duration"),
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: &ast.StarExpr{
							X: ast.NewIdent("Store"),
						},
					},
				},
			},
		},
		// Function body
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				// s := Store{interval: interval, segmentRepo: segmentRepo}
				&ast.AssignStmt{
					Lhs: []ast.Expr{ast.NewIdent("s")},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.CompositeLit{
							Type: ast.NewIdent("Store"),
							Elts: []ast.Expr{
								&ast.KeyValueExpr{
									Key:   ast.NewIdent("interval"),
									Value: ast.NewIdent("interval"),
								},
								&ast.KeyValueExpr{
									Key:   ast.NewIdent("repo"),
									Value: ast.NewIdent("repo"),
								},
							},
						},
					},
				},
				// s.load()
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   ast.NewIdent("s"),
							Sel: ast.NewIdent("load"),
						},
					},
				},
				// go s.loop()
				&ast.GoStmt{
					Call: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   ast.NewIdent("s"),
							Sel: ast.NewIdent("loop"),
						},
					},
				},
				// return &s
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.UnaryExpr{
							Op: token.AND,
							X:  ast.NewIdent("s"),
						},
					},
				},
			},
		},
	}
	return newFunc
}

func (i *Implementator) loopFunc() ast.Decl {
	loopFunc := &ast.FuncDecl{
		Name: ast.NewIdent("loop"),
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent("s")},
					Type: &ast.StarExpr{
						X: ast.NewIdent("Store"),
					},
				},
			},
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
		},
		// Function body
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				// for {
				&ast.ForStmt{
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							// time.Sleep(s.interval)
							&ast.ExprStmt{
								X: &ast.CallExpr{
									Fun: &ast.SelectorExpr{
										X:   ast.NewIdent("time"),
										Sel: ast.NewIdent("Sleep"),
									},
									Args: []ast.Expr{
										ast.NewIdent("s.interval"),
									},
								},
							},
							// s.load()
							&ast.ExprStmt{
								X: &ast.CallExpr{
									Fun: &ast.SelectorExpr{
										X:   ast.NewIdent("s"),
										Sel: ast.NewIdent("load"),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return loopFunc
}

func (i *Implementator) loadFunc() ast.Decl {
	return &ast.FuncDecl{
		Name: ast.NewIdent("load"),
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent("s")},
					Type: &ast.StarExpr{
						X: ast.NewIdent("Store"),
					},
				},
			},
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
		},
		// Function body
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				// segments, _, err := s.segmentRepo.GetSegments()
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						ast.NewIdent(i.variableName),
						ast.NewIdent("err"),
					},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X: &ast.SelectorExpr{
									X:   ast.NewIdent("s"),
									Sel: ast.NewIdent("repo"),
								},
								Sel: ast.NewIdent(i.funcName),
							},
						},
					},
				},
				// if err != nil {
				&ast.IfStmt{
					Cond: &ast.BinaryExpr{
						X:  ast.NewIdent("err"),
						Op: token.NEQ,
						Y:  ast.NewIdent("nil"),
					},
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							// slog.Error("loading segments failed", "err", err)
							&ast.ExprStmt{
								X: &ast.CallExpr{
									Fun: &ast.SelectorExpr{
										X:   ast.NewIdent("log"),
										Sel: ast.NewIdent("Println"),
									},
									Args: []ast.Expr{
										ast.NewIdent(`"loading store failed"`),
										ast.NewIdent("err"),
									},
								},
							},
							// return
							&ast.ReturnStmt{},
						},
					},
				},
				// s.segmentLock.Lock()
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X: &ast.SelectorExpr{
								X:   ast.NewIdent("s"),
								Sel: ast.NewIdent(i.lockName),
							},
							Sel: ast.NewIdent("Lock"),
						},
					},
				},
				// s.segments = segments
				&ast.AssignStmt{
					Lhs: []ast.Expr{ast.NewIdent("s." + i.variableName)},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{ast.NewIdent(i.variableName)},
				},
				// s.segmentLock.Unlock()
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X: &ast.SelectorExpr{
								X:   ast.NewIdent("s"),
								Sel: ast.NewIdent(i.lockName),
							},
							Sel: ast.NewIdent("Unlock"),
						},
					},
				},
			},
		},
	}
}

func (i *Implementator) implementFunction() ast.Decl {
	return &ast.FuncDecl{
		Name: ast.NewIdent(i.funcName),
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent("s")},
					Type: &ast.StarExpr{
						X: ast.NewIdent("Store"),
					},
				},
			},
		},
		Type: &ast.FuncType{
			Results: i.methodDef.Type.(*ast.FuncType).Results,
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				// s.segmentLock.RLock()
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X: &ast.SelectorExpr{
								X:   ast.NewIdent("s"),
								Sel: ast.NewIdent(i.lockName),
							},
							Sel: ast.NewIdent("RLock"),
						},
					},
				},
				// defer s.segmentLock.RUnlock()
				&ast.DeferStmt{
					Call: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X: &ast.SelectorExpr{
								X:   ast.NewIdent("s"),
								Sel: ast.NewIdent(i.lockName),
							},
							Sel: ast.NewIdent("RUnlock"),
						},
					},
				},
				// return s.segments, nil, nil
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.SelectorExpr{
							X:   ast.NewIdent("s"),
							Sel: ast.NewIdent(i.variableName),
						},
						ast.NewIdent("nil"),
					},
				},
			},
		},
	}
}
