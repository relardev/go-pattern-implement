package filterreturn

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
	return "filter-return"
}

func (i *Implementator) Description() string {
	return "Filter collection that is returned using list of given functions"
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
			validate(methodDef)

			filterFuncsSignature := text.ToExpr(fstr.Sprintf(map[string]any{
				"params": getBaseType(methodDef.Type.(*ast.FuncType).Results.List[0].Type),
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

	resultVars := naming.ExtractFuncReturns(field)

	finalReturns := make([]ast.Expr, len(resultVars))
	for i, r := range resultVars {
		if i == 0 {
			finalReturns[i] = ast.NewIdent("filtered")
			continue
		}
		finalReturns[i] = r
	}

	t := fstr.Sprintf(map[string]any{
		"firstLetter": unicode.ToLower(rune(i.interfaceName[0])),
		"fnName":      field.Names[0].Name,
		"args": code.AddPackageNameToFieldList(
			field.Type.(*ast.FuncType).Params,
			i.packageName,
		),
		"results":          results,
		"varArgs":          naming.ExtractFuncArgs(field),
		"resultType":       field.Type.(*ast.FuncType).Results.List[0].Type,
		"resultVars":       resultVars,
		"resultVar":        resultVars[0],
		"addToFilterered":  appendOrSet(results.List[0].Type, "filtered", "item"),
		"rangeDestructure": rangeDestructure(results.List[0].Type, "item"),
		"return":           finalReturns,
	}, `
func ({{firstLetter}} *Filter) {{fnName}}({{args}}) ({{results}}) {
	{{resultVars}} := {{firstLetter}}.{{firstLetter}}.{{fnName}}({{varArgs}})
	filtered := {{resultType}}{}
OUTER:
	for {{rangeDestructure}} := range {{resultVar}} {
		for _, filter := range {{firstLetter}}.filters {
			if !filter(item) {
			 	continue OUTER
			}
		}
		{{addToFilterered}}
	}
	return {{return}}
}`)

	return text.ToDecl(t)
}

func validate(field *ast.Field) {
	returns := field.Type.(*ast.FuncType).Results
	if returns == nil || len(returns.List) == 0 {
		panic("Expected some returns")
	}

	enumerable := returns.List[0].Type
	if !code.IsEnumerable(enumerable) {
		panic("Expected enumerable as first return")
	}
}

func getBaseType(retType ast.Expr) ast.Expr {
	switch t := retType.(type) {
	case *ast.ArrayType:
		return t.Elt
	case *ast.MapType:
		return t.Value
	case *ast.StarExpr:
		return getBaseType(t.X)
	default:
		panic("unsupported type")
	}
}

func appendOrSet(t ast.Expr, varName, valueName string) string {
	switch t := t.(type) {
	case *ast.ArrayType:
		return fmt.Sprintf("%s = append(%s, %s)", varName, varName, valueName)
	case *ast.MapType:
		return fmt.Sprintf("%s[key] = %s", varName, valueName)
	case *ast.StarExpr:
		return appendOrSet(t.X, varName, valueName)
	default:
		panic("unsupported type")
	}
}

func rangeDestructure(t ast.Expr, varName string) string {
	switch t := t.(type) {
	case *ast.ArrayType:
		return fmt.Sprintf("_, %s", varName)
	case *ast.MapType:
		return fmt.Sprintf("key, %s", varName)
	case *ast.StarExpr:
		return rangeDestructure(t.X, varName)
	default:
		panic("unsupported type")
	}
}
