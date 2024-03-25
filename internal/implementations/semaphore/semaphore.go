package semaphore

import (
	"component-generator/internal/code"
	"component-generator/internal/fstr"
	"component-generator/internal/text"
	"fmt"
	"go/ast"
	"unicode"
)

type Implementator struct {
	err           error
	packageName   string
	interfaceName string
}

func New(sourcePackageName string) *Implementator {
	return &Implementator{
		packageName: sourcePackageName,
	}
}

func (i *Implementator) Name() string {
	return "semaphore"
}

func (i *Implementator) Description() string {
	return "Simple semaphore implementation"
}

func (i *Implementator) Error() error {
	return i.err
}

func (i *Implementator) Visit(node ast.Node) (bool, []ast.Decl) {
	decls := []ast.Decl{}

	switch typeSpec := node.(type) {
	case *ast.TypeSpec:
		i.interfaceName = typeSpec.Name.Name
		decls = append(decls, code.Struct(
			"Semaphore",
			code.FieldFromTypeSpec(typeSpec, i.packageName),
			code.StructField{
				Name:    "c",
				TypeStr: "chan struct{}",
			},
		))
		decls = append(decls, i.newWraperFunction())

		switch interfaceNode := typeSpec.Type.(type) {
		case *ast.InterfaceType:
			for _, methodDef := range interfaceNode.Methods.List {
				decls = append(decls, i.implementFunction(methodDef))
			}
		default:
			panic("not an interface")
		}
	default:
		return true, nil
	}

	return false, decls
}

func (i *Implementator) newWraperFunction() ast.Decl {
	template := fstr.Sprintf(map[string]any{
		"firstLetter":       unicode.ToLower(rune(i.interfaceName[0])),
		"interfaceSelector": fmt.Sprintf("%s.%s", i.packageName, i.interfaceName),
	}, `
	func New({{firstLetter}} {{interfaceSelector}}, allowedParallelExecutions int) *Semaphore {
		return &Semaphore{
			{{firstLetter}}: {{firstLetter}},
			c:	make(chan struct{}, allowedParallelExecutions),
		}
	}`)

	return text.ToDecl(template)
}

func (i *Implementator) implementFunction(field *ast.Field) ast.Decl {
	takesContext := code.IsContext(field.Type.(*ast.FuncType).Params.List[0].Type)

	results := code.AddPackageNameToResults(field.Type.(*ast.FuncType).Results, i.packageName)

	commonArgs := map[string]any{
		"firstLetter": unicode.ToLower(rune(i.interfaceName[0])),
		"fnName":      field.Names[0].Name,
		"args":        field.Type.(*ast.FuncType).Params,
		"varArgs":     code.ExtractFuncArgs(field),
		"results":     results,
	}

	var t string
	if takesContext {
		var zeroReturns []ast.Expr
		for _, r := range field.Type.(*ast.FuncType).Results.List {
			zeroReturns = append(zeroReturns, code.ZeroValue(r.Type))
		}

		returnsError, errorPos := code.DoesReturnError(field)
		if returnsError {
			zeroReturns[errorPos] = text.ToExpr("ctx.Err()")
		}

		commonArgs["zeroReturns"] = zeroReturns

		t = fstr.Sprintf(
			commonArgs,
			`
func (s *Semaphore) {{fnName}}({{args}}) ({{results}}) {
	select {
	case s.c <- struct{}{}:
		defer func() { <-s.c }()
		return s.{{firstLetter}}.{{fnName}}({{varArgs}})
	case <-ctx.Done():
		return {{zeroReturns}}
	}
}`)
	} else {
		t = fstr.Sprintf(commonArgs, `
func (s *Semaphore) {{fnName}}({{args}}) ({{results}}) {
	s.c <- struct{}{} 
	defer func() { <-s.c }()
	return s.{{firstLetter}}.{{fnName}}({{varArgs}})
}`)
	}

	return text.ToDecl(t)
}
