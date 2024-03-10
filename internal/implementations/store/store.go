package store

import (
	"go/ast"
	"go/token"

	"component-generator/internal/code"
	"component-generator/internal/naming"
)

type NewBehaviour bool

const (
	PanicInNew       NewBehaviour = true
	ReturnErrorInNew NewBehaviour = false
)

type Implementator struct {
	// args
	packageName  string
	newBehaviour NewBehaviour

	// local vars
	err           error
	interfaceName string
	methodDef     *ast.Field
	funcName      string
	variableName  string
	lockName      string
	argIsContext  bool
}

func New(sourcePackageName string, panicInNew NewBehaviour) *Implementator {
	return &Implementator{
		packageName:  sourcePackageName,
		newBehaviour: panicInNew,
	}
}

func (i *Implementator) Name() string {
	if i.newBehaviour {
		return "store-panic"
	} else {
		return "store-err"
	}
}

func (i *Implementator) Description() string {
	base := "store for rarely changeing data, that you want to have in memory"
	if i.newBehaviour {
		return base + " (panics in New)"
	} else {
		return base + " (returns error in New)"
	}
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

			args := i.methodDef.Type.(*ast.FuncType).Params.List

			if len(args) == 1 {
				i.argIsContext = code.IsContext(args[0].Type)
			}

			if !(len(args) == 0 || i.argIsContext) {
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
			decls = append(decls, i.newFunc())
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

func (i *Implementator) newFunc() ast.Decl {
	returns := []*ast.Field{
		{
			Type: &ast.StarExpr{
				X: ast.NewIdent("Store"),
			},
		},
	}

	iferr := code.IfErrReturnErr()

	switch i.newBehaviour {
	case PanicInNew:
		iferr.Body.List[0] = &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: ast.NewIdent("panic"),
				Args: []ast.Expr{
					ast.NewIdent("err"),
				},
			},
		}

	case ReturnErrorInNew:
		returns = append(returns, &ast.Field{
			Type: ast.NewIdent("error"),
		})
	}

	newFunc := &ast.FuncDecl{
		Name: ast.NewIdent("New"),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("repo")},
						Type:  ast.NewIdent(i.interfaceName),
					},
					{
						Names: []*ast.Ident{ast.NewIdent("interval")},
						Type:  ast.NewIdent("time.Duration"),
					},
				},
			},
			Results: &ast.FieldList{
				List: returns,
			},
		},
		// Function body
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				// s := Store{segmentRepo: segmentRepo}
				&ast.AssignStmt{
					Lhs: []ast.Expr{ast.NewIdent("s")},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.CompositeLit{
							Type: ast.NewIdent("Store"),
							Elts: []ast.Expr{
								&ast.KeyValueExpr{
									Key:   ast.NewIdent("repo"),
									Value: ast.NewIdent("repo"),
								},
							},
						},
					},
				},
				// err := s.load()
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						ast.NewIdent("err"),
					},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent("s"),
								Sel: ast.NewIdent("load"),
							},
						},
					},
				},
				iferr,
				// go s.loop(interval)
				&ast.GoStmt{
					Call: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   ast.NewIdent("s"),
							Sel: ast.NewIdent("loop"),
						},
						Args: []ast.Expr{
							ast.NewIdent("interval"),
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
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("interval")},
						Type:  ast.NewIdent("time.Duration"),
					},
				},
			},
		},
		// Function body
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				// for {
				&ast.ForStmt{
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							// time.Sleep(interval)
							&ast.ExprStmt{
								X: &ast.CallExpr{
									Fun: &ast.SelectorExpr{
										X:   ast.NewIdent("time"),
										Sel: ast.NewIdent("Sleep"),
									},
									Args: []ast.Expr{
										ast.NewIdent("interval"),
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
	interfaceMethodCallArg := []ast.Expr{}
	if i.argIsContext {
		interfaceMethodCallArg = append(
			interfaceMethodCallArg,
			ast.NewIdent("context.Background()"),
		)
	}
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
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: ast.NewIdent("error"),
					},
				},
			},
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
							Args: interfaceMethodCallArg,
						},
					},
				},
				code.IfErrReturnErr(),
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
	params := i.methodDef.Type.(*ast.FuncType).Params.List
	if i.argIsContext {
		params[0] = &ast.Field{
			Names: []*ast.Ident{ast.NewIdent("_")},
			Type:  ast.NewIdent("context.Context"),
		}
	}
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
			Params:  &ast.FieldList{List: params},
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
