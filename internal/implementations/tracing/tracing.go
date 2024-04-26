package tracing

import (
	"fmt"
	"go/ast"
	"unicode"

	"github.com/relardev/go-pattern-implement/internal/code"
	"github.com/relardev/go-pattern-implement/internal/fstr"
	"github.com/relardev/go-pattern-implement/internal/naming"
	"github.com/relardev/go-pattern-implement/internal/text"
)

type Implementator struct {
	packageName string

	interfaceName string
}

func New(sourcePackageName string) *Implementator {
	return &Implementator{
		packageName: sourcePackageName,
	}
}

func (i *Implementator) Name() string {
	return "tracing"
}

func (i *Implementator) Description() string {
	return "Generate traceing wrapper"
}

func (i *Implementator) Error() error {
	return nil
}

func (i *Implementator) Visit(node ast.Node) (bool, []ast.Decl) {
	decls := []ast.Decl{}

	switch typeSpec := node.(type) {
	case *ast.TypeSpec:
		i.interfaceName = typeSpec.Name.Name
		decls = append(decls,
			code.Struct(
				i.interfaceName+"Tracer",
				code.FieldFromTypeSpec(typeSpec, i.packageName),
				code.StructField{
					Name:    "tracer",
					TypeStr: "trace.Tracer",
				},
			))
		decls = append(decls, i.newWrapperFunction())

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

func (i *Implementator) newWrapperFunction() ast.Decl {
	template := fstr.Sprintf(map[string]any{
		"interfaceName":     i.interfaceName,
		"firstLetter":       unicode.ToLower(rune(i.interfaceName[0])),
		"interfaceSelector": fmt.Sprintf("%s.%s", i.packageName, i.interfaceName),
		"pkgName":           i.packageName,
	}, `
	func New{{interfaceName}}({{firstLetter}} {{interfaceSelector}}) *{{interfaceName}}Tracer {
		return &{{interfaceName}}Tracer{
			{{firstLetter}}: {{firstLetter}},
			tracer:      otel.Tracer("{{pkgName}}.{{interfaceName}}"),
		}
	}`)

	return text.ToDecl(template)
}

func (i *Implementator) implementFunction(interfaceName string, field *ast.Field) ast.Decl {
	i.validate(field)
	results := code.AddPackageNameToFieldList(field.Type.(*ast.FuncType).Results, i.packageName)

	varArgs := naming.ExtractFuncArgs(field)

	varArgs[0] = ast.NewIdent("spanCtx")

	template := fstr.Sprintf(map[string]any{
		"interfaceName": i.interfaceName,
		"firstLetter":   unicode.ToLower(rune(i.interfaceName[0])),
		"fnName":        field.Names[0].Name,
		"args":          field.Type.(*ast.FuncType).Params,
		"results":       results,
		"varArgs":       varArgs,
		"resultVars":    naming.ExtractFuncReturns(field),
		"traceName":     fmt.Sprintf("%s.%s", interfaceName, field.Names[0].Name),
	}, `
func (t *{{interfaceName}}Tracer) {{fnName}}({{args}}) ({{results}}) {
	spanCtx, span := t.tracer.Start(ctx, "{{traceName}}")
	defer span.End()

	{{resultVars}} := t.{{firstLetter}}.{{fnName}}({{varArgs}})

	if err != nil {
		span.SetStatus(
			codes.Error,
			"{{traceName}} failed",
		)
		span.RecordError(err)

		return {{resultVars}}
	}

	span.AddEvent("{{traceName}} succeded")

	return {{resultVars}}
}
`)
	return text.ToDecl(template)
}

func (i *Implementator) validate(field *ast.Field) {
	if !code.IsContext(field.Type.(*ast.FuncType).Params.List[0].Type) {
		panic("first argument must be a context")
	}
	returnsError, _ := code.DoesFieldReturnError(field)
	if !returnsError {
		panic("last return value must be an error")
	}
}
