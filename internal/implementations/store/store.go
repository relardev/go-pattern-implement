package store

import (
	"go/ast"
	"go/token"

	"component-generator/internal/code"
	"component-generator/internal/naming"
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
			methodDef := interfaceNode.Methods.List[0]

			returns := methodDef.Type.(*ast.FuncType).Results.List

			if len(returns) != 1 && len(returns) != 2 {
				panic("method should have one or two returns")
			}

			resultName := naming.VariableNameFromExpr(returns[0].Type)
			resultType := "[]*model.Segment"

			field := code.FieldFromTypeSpec(typeSpec, i.packageName)

			fields := []code.StructField{
				{
					Name: "repo",
					Type: field.Type,
				},
				{
					Name: "interval",
					Type: "time.Duration",
				},
				{
					Name: resultName + "Lock",
					Type: "sync.RWMutex",
				},
				{
					Name: resultName,
					Type: resultType,
				},
			}
			decls = append(decls, code.Struct("Store", fields...))
			decls = append(decls, newFunc(typeSpec.Name.Name, i.packageName))
			decls = append(decls, loopFunc(typeSpec.Name.Name, i.packageName))
			decls = append(decls, loadFunc(typeSpec.Name.Name, i.packageName))

			for range interfaceNode.Methods.List {
				// decls = append(decls, i.implementFunction(typeSpec.Name.Name, methodDef))
			}
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
						Type: &ast.SelectorExpr{
							X:   ast.NewIdent("repository"),
							Sel: ast.NewIdent("Segment"),
						},
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

func loopFunc(interfaceName, interfacePackage string) ast.Decl {
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

// Generate function like this:
//
//	func (s *Store) load() {
//		segments, _, err := s.segmentRepo.GetSegments()
//		if err != nil {
//			slog.Error("loading segments failed", "err", err)
//			return
//		}
//
//		s.segmentLock.Lock()
//		s.segments = segments
//		s.segmentLock.Unlock()
//	}
func loadFunc(interfaceName, interfacePackage string) ast.Decl {
	loadFunc := &ast.FuncDecl{
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
						ast.NewIdent("segments"),
						ast.NewIdent("_"),
						ast.NewIdent("err"),
					},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X: &ast.SelectorExpr{
									X:   ast.NewIdent("s"),
									Sel: ast.NewIdent("segmentRepo"),
								},
								Sel: ast.NewIdent("GetSegments"),
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
										X:   ast.NewIdent("slog"),
										Sel: ast.NewIdent("Error"),
									},
									Args: []ast.Expr{
										ast.NewIdent(`"loading segments failed"`),
										ast.NewIdent(`"err"`),
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
								Sel: ast.NewIdent("segmentLock"),
							},
							Sel: ast.NewIdent("Lock"),
						},
					},
				},
				// s.segments = segments
				&ast.AssignStmt{
					Lhs: []ast.Expr{ast.NewIdent("s.segments")},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{ast.NewIdent("segments")},
				},
				// s.segmentLock.Unlock()
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X: &ast.SelectorExpr{
								X:   ast.NewIdent("s"),
								Sel: ast.NewIdent("segmentLock"),
							},
							Sel: ast.NewIdent("Unlock"),
						},
					},
				},
			},
		},
	}
	return loadFunc
}
