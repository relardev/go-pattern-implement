package cache

import (
	"component-generator/internal/code"
	"component-generator/internal/fstr"
	"component-generator/internal/naming"
	"component-generator/internal/text"
	"fmt"
	"go/ast"
	"unicode"
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
	return "cache"
}

func (i *Implementator) Description() string {
	return "Cache results of wrapped interface"
}

func (i *Implementator) Error() error {
	return i.err
}

func (i *Implementator) Visit(node ast.Node) (bool, []ast.Decl) {
	decls := []ast.Decl{}

	switch typeSpec := node.(type) {
	case *ast.TypeSpec:
		decls = append(decls, code.Struct(
			"Cache",
			code.FieldFromTypeSpec(typeSpec, i.packageName),
			code.StructField{
				Name:    "cache",
				TypeStr: "*cache.Cache",
			},
		))
		decls = append(decls, newWraperFunction(typeSpec.Name.Name, i.packageName))

		switch interfaceNode := typeSpec.Type.(type) {
		case *ast.InterfaceType:
			for _, methodDef := range interfaceNode.Methods.List {
				validateReturnList(methodDef.Type.(*ast.FuncType).Results.List)
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

func validateReturnList(returnList []*ast.Field) {
	if len(returnList) != 2 {
		panic("only methods with 2 return values are supported")
	}

	// check if last return value is error
	if !code.IsError(returnList[1].Type) {
		panic("last return value must be an error")
	}
}

func newWraperFunction(interfaceName, interfacePackage string) ast.Decl {
	firstLetter := unicode.ToLower(rune(interfaceName[0]))

	wrappedName := fmt.Sprintf("%s.%s", interfacePackage, interfaceName)

	template := fmt.Sprintf(`
	func New(%s %s, expiration, cleanupInterval time.Duration) *Cache {
		return &%s{
			%s: %s,
			cache: cache.New(expiration, cleanupInterval),
		}
	}
	`, string(firstLetter),
		wrappedName,
		interfaceName,
		string(firstLetter),
		string(firstLetter),
	)

	return text.ToDecl(template)
}

func (i *Implementator) implementFunction(interfaceName string, field *ast.Field) ast.Decl {
	result := field.Type.(*ast.FuncType).Results.List[0]
	result.Type = code.PossiblyAddPackageName(i.packageName, result.Type)

	t := fstr.Sprintf(map[string]any{
		"firstLetter": unicode.ToLower(rune(interfaceName[0])),
		"fnName":      field.Names[0].Name,
		"args":        field.Type.(*ast.FuncType).Params,
		"varType":     result,
		"varName":     naming.VariableNameFromExpr(result.Type),
		"zeroValue":   code.ZeroValue(result.Type),
		"varArgs":     code.ExtractFuncArgs(field),
	}, `
func ({{firstLetter}} *Cache) {{fnName}}({{args}}) ({{varType}}, error) {
	key := ""
	cachedItem, found := {{firstLetter}}.cache.Get(key)
	if found {
		{{varName}}, ok := cachedItem.({{varType}})
		if !ok {
			return {{zeroValue}}, errors.New("invalid object in cache")
		}
		return {{varName}}, nil
	}
	{{varName}} := {{firstLetter}}.{{firstLetter}}.{{fnName}}({{varArgs}})
	if err != nil {
		return {{zeroValue}}, err
	}

	{{firstLetter}}.cache.Set(key, {{varName}}, cache.DefaultExpiration)

	return {{varName}}, nil
}`)

	return text.ToDecl(t)
}
