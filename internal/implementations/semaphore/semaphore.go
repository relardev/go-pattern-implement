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
	hasContext := code.IsContext(field.Type.(*ast.FuncType).Params.List[0].Type)
	result := field.Type.(*ast.FuncType).Results.List[0]
	result.Type = code.PossiblyAddPackageName(i.packageName, result.Type)

	var contextPart string
	if hasContext {
		contextPart = `
	case <-ctx.Done():
		return ctx.Err()
`
	}

	t := fstr.Sprintf(map[string]any{
		"firstLetter": unicode.ToLower(rune(i.interfaceName[0])),
		"fnName":      field.Names[0].Name,
		"args":        field.Type.(*ast.FuncType).Params,
		// "varType":     result,
		// "varName": naming.VariableNameFromExpr(result.Type),
		// "zeroValue":   code.ZeroValue(result.Type),
		"varArgs":     code.ExtractFuncArgs(field),
		"contextPart": contextPart,
	}, `
func (s *Semaphore) Materialize({{args}}) error {
	select {
	case s.c <- struct{}{}:
		defer func() { <-s.c }()
		return s.{{firstLetter}}.{{fnName}}({{varArgs}})
	{{contextPart}}
	}
}`)

	return text.ToDecl(t)
}
