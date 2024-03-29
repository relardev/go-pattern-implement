package filterparam

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
	addContext    bool
}

func New(sourcePackageName string) *Implementator {
	return &Implementator{
		packageName: sourcePackageName,
	}
}

func (i *Implementator) Name() string {
	return "filter-param"
}

func (i *Implementator) Description() string {
	return "Filter collection that is passed by function parameters using list of given functions"
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
			if code.IsContext(methodDef.Type.(*ast.FuncType).Params.List[0].Type) {
				i.addContext = true
				methodDef.Type.(*ast.FuncType).Params.List = methodDef.Type.(*ast.FuncType).Params.List[1:]
			}

			validate(methodDef)
			params := code.AddPackageNameToFieldList(methodDef.Type.(*ast.FuncType).Params, i.packageName)

			filterFuncsSignature := text.ToExpr(fstr.Sprintf(map[string]any{
				"params": getBaseType(params.List[0].Type),
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
	params := field.Type.(*ast.FuncType).Params

	paramType := params.List[0].Type
	addToFiltered := appendOrSet(paramType, "filtered", "item")
	rangeDestruct := rangeDestructure(paramType, "item")

	paramVars := naming.ExtractFuncArgs(field)

	finalParams := make([]ast.Expr, len(paramVars))
	for i, r := range paramVars {
		if i == 0 {
			finalParams[i] = ast.NewIdent("filtered")
			continue
		}
		finalParams[i] = r
	}

	if i.addContext {
		params.List = append(
			[]*ast.Field{
				{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: ast.NewIdent("context.Context")},
			},
			params.List...,
		)
		finalParams = append([]ast.Expr{ast.NewIdent("ctx")}, finalParams...)
	}

	results := field.Type.(*ast.FuncType).Results

	var returnText string
	if results != nil && len(results.List) != 0 {
		returnText = "return"
	}

	t := fstr.Sprintf(map[string]any{
		"firstLetter":      unicode.ToLower(rune(i.interfaceName[0])),
		"fnName":           field.Names[0].Name,
		"params":           params,
		"results":          field.Type.(*ast.FuncType).Results,
		"varArgs":          finalParams,
		"paramsType":       paramType,
		"paramVar":         paramVars[0],
		"addToFilterered":  addToFiltered,
		"rangeDestructure": rangeDestruct,
		"return":           returnText,
	}, `
func ({{firstLetter}} *Filter) {{fnName}}({{params}}) ({{results}}) {
	filtered := {{paramsType}}{}
OUTER:
	for {{rangeDestructure}} := range {{paramVar}} {
		for _, filter := range {{firstLetter}}.filters {
			if !filter(item) {
			 	continue OUTER
			}
		}
		{{addToFilterered}}
	}
	{{return}} {{firstLetter}}.{{firstLetter}}.{{fnName}}({{varArgs}})
}`)

	return text.ToDecl(t)
}

func validate(field *ast.Field) {
	params := field.Type.(*ast.FuncType).Params
	if params == nil || len(params.List) == 0 {
		panic("Expected some params")
	}

	enumerable := params.List[0].Type
	if !code.IsEnumerable(enumerable) {
		panic("Expected enumerable as parameter")
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
