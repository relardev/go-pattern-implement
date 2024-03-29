package filter

import (
	"fmt"
	"go/ast"
	"unicode"

	"github.com/relardev/go-pattern-implement/internal/code"
	"github.com/relardev/go-pattern-implement/internal/fstr"
	"github.com/relardev/go-pattern-implement/internal/naming"
	"github.com/relardev/go-pattern-implement/internal/text"
)

type Mode int

const (
	ModeNoError Mode = iota
	ModeWithError
)

type Implementator struct {
	err           error
	packageName   string
	interfaceName string
	mode          Mode
}

func New(sourcePackageName string, m Mode) *Implementator {
	return &Implementator{
		packageName: sourcePackageName,
		mode:        m,
	}
}

func (i *Implementator) Name() string {
	if i.mode == ModeNoError {
		return "filter"
	}

	return "filter-error"
}

func (i *Implementator) Description() string {
	if i.mode == ModeNoError {
		return "Stop processing call if any of the filter functions return false, don't return error"
	}

	return "Stop processing call if any of the filter functions return false, returns error"
}

func (i *Implementator) Error() error {
	return i.err
}

func (i *Implementator) Visit(node ast.Node) (bool, []ast.Decl) {
	decls := []ast.Decl{}

	switch typeSpec := node.(type) {
	case *ast.TypeSpec:
		i.interfaceName = typeSpec.Name.Name
		switch interfaceNode := typeSpec.Type.(type) {
		case *ast.InterfaceType:
			if len(interfaceNode.Methods.List) != 1 {
				panic("expected exactly one method")
			}
			methodDef := interfaceNode.Methods.List[0]
			i.validate(methodDef)

			filterFuncsSignature := text.ToExpr(fstr.Sprintf(map[string]any{
				"params": code.RemoveNames(methodDef.Type.(*ast.FuncType).Params),
			},
				"[]func({{params}}) bool",
			))
			decls = append(decls, code.Struct(
				"Filter",
				code.FieldFromTypeSpec(typeSpec, i.packageName),
				code.StructField{
					Name:     "filters",
					TypeSpec: filterFuncsSignature,
				},
			))
			decls = append(decls, i.newWraperFunction(filterFuncsSignature))
			decls = append(decls, i.implementFunction(methodDef))

		default:
			panic("not an interface")
		}
	default:
		return true, nil
	}

	return false, decls
}

func (i *Implementator) newWraperFunction(filtersSigature ast.Expr) ast.Decl {
	template := fstr.Sprintf(map[string]any{
		"firstLetter":       unicode.ToLower(rune(i.interfaceName[0])),
		"interfaceSelector": fmt.Sprintf("%s.%s", i.packageName, i.interfaceName),
		"filtersSigs":       filtersSigature,
	}, `
	func New({{firstLetter}} {{interfaceSelector}}, filters {{filtersSigs}}) *Filter {
		return &Filter{
			{{firstLetter}}: {{firstLetter}},
			filters: filters,
		}
	}`)

	return text.ToDecl(template)
}

func (i *Implementator) implementFunction(field *ast.Field) ast.Decl {
	results := code.AddPackageNameToFieldList(field.Type.(*ast.FuncType).Results, i.packageName)

	zeroReturns := i.getReturns(results)

	returnPartArgs := map[string]any{
		"firstLetter": unicode.ToLower(rune(i.interfaceName[0])),
		"fnName":      field.Names[0].Name,
		"varArgs":     naming.ExtractFuncArgs(field),
	}

	var returnPart string
	if results != nil {
		returnPart = fstr.Sprintf(
			returnPartArgs,
			"return {{firstLetter}}.{{fnName}}({{varArgs}})",
		)
	} else {
		returnPart = fstr.Sprintf(
			returnPartArgs,
			`{{firstLetter}}.{{fnName}}({{varArgs}})
	return`,
		)
	}

	t := fstr.Sprintf(map[string]any{
		"firstLetter": unicode.ToLower(rune(i.interfaceName[0])),
		"fnName":      field.Names[0].Name,
		"args":        field.Type.(*ast.FuncType).Params,
		"results":     results,
		"zeroReturns": zeroReturns,
		"returnPart":  returnPart,
		"varArgs":     naming.ExtractFuncArgs(field),
	}, `
func ({{firstLetter}} *Filter) {{fnName}}({{args}}) ({{results}}) {
	for _, filter := range {{firstLetter}}.filters {
		if !filter({{varArgs}}) {
			return {{zeroReturns}}
		}
	}
	{{returnPart}}
}`)

	return text.ToDecl(t)
}

func (i *Implementator) getReturns(results *ast.FieldList) []ast.Expr {
	if results == nil {
		return nil
	}

	if i.mode == ModeNoError {
		// only possible return can be an error
		return []ast.Expr{
			ast.NewIdent("nil"),
		}
	}

	return []ast.Expr{
		text.ToExpr("errors.New(\"filtered\")"),
	}
}

func (i *Implementator) validate(field *ast.Field) {
	returns := field.Type.(*ast.FuncType).Results
	if i.mode == ModeWithError {
		if returns == nil || len(returns.List) != 1 || !code.IsError(returns.List[0].Type) {
			panic("expected error as the only return value")
		}
		return
	}
	if returns == nil || len(returns.List) == 0 {
		return
	}

	if len(returns.List) == 1 {
		if !code.IsError(returns.List[0].Type) {
			panic("expected error as the only return value")
		}

		return
	}

	panic("expected 1 or 0 return values")
}
