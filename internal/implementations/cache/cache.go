package cache

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
	template := fstr.Sprintf(map[string]any{
		"firstLetter":       unicode.ToLower(rune(interfaceName[0])),
		"interfaceSelector": fmt.Sprintf("%s.%s", interfacePackage, interfaceName),
	}, `
	func New({{firstLetter}} {{interfaceSelector}}, expiration, cleanupInterval time.Duration) *Cache {
		return &Cache{
			{{firstLetter}}: {{firstLetter}},
			cache: cache.New(expiration, cleanupInterval),
		}
	}`)

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
		"varArgs":     naming.ExtractFuncArgs(field),
		"key":         generateKey(field.Type.(*ast.FuncType).Params),
	}, `
func ({{firstLetter}} *Cache) {{fnName}}({{args}}) ({{varType}}, error) {
	key := {{key}}
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

func generateKey(params *ast.FieldList) string {
	key := "\"TODO\""
	withoutContext := []*ast.Field{}
	if code.IsContext(params.List[0].Type) {
		for i := 1; i < len(params.List); i++ {
			withoutContext = append(withoutContext, params.List[i])
		}
	} else {
		withoutContext = params.List
	}

	if len(withoutContext) == 1 && len(withoutContext[0].Names) == 1 {
		nodeType := withoutContext[0].Type
		switch t := nodeType.(type) {
		case *ast.Ident:
			if t.Name == "string" {
				return withoutContext[0].Names[0].Name
			}
		default:
			return key
		}
	}
	return key
}
