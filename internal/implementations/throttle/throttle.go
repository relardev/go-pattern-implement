package throttle

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
	return "throttle"
}

func (i *Implementator) Description() string {
	return "Process at most n requests per second"
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
			"Throttle",
			code.FieldFromTypeSpec(typeSpec, i.packageName),
			code.StructField{
				Name:    "ticker",
				TypeStr: "*time.Ticker",
			},
			code.StructField{
				Name:    "mu",
				TypeStr: "sync.Mutex",
			},
			code.StructField{
				Name:    "alreadyCalled",
				TypeStr: "bool",
			},
		))
		decls = append(decls, i.newWraperFunction())
		decls = append(decls, i.resetCounterFunction())

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
	func New({{firstLetter}} {{interfaceSelector}}, passesPerSecond int) *Throttle {
		throttle := &Throttle{
			{{firstLetter}}: {{firstLetter}},
			ticker:    time.NewTicker(time.Second / time.Duration(passesPerSecond)),
		}

		go throttle.resetCounter()
		return throttle
	}`)

	return text.ToDecl(template)
}

func (i *Implementator) resetCounterFunction() ast.Decl {
	template := fstr.Sprintf(map[string]any{
		"firstLetter": unicode.ToLower(rune(i.interfaceName[0])),
	}, `
func ({{firstLetter}} *Throttle) resetCounter() {
	for range p.ticker.C {
		p.mu.Lock()
		p.alreadyCalled = false
		p.mu.Unlock()
	}
}
`)

	return text.ToDecl(template)
}

func (i *Implementator) implementFunction(field *ast.Field) ast.Decl {
	results := field.Type.(*ast.FuncType).Results
	if results != nil {
		for j, r := range results.List {
			results.List[j].Type = code.PossiblyAddPackageName(i.packageName, r.Type)
		}
	}

	t := fstr.Sprintf(map[string]any{
		"firstLetter": unicode.ToLower(rune(i.interfaceName[0])),
		"fnName":      field.Names[0].Name,
		"args":        field.Type.(*ast.FuncType).Params,
		"varArgs":     code.ExtractFuncArgs(field),
		"results":     results,
	}, `
func ({{firstLetter}} *Throttle) {{fnName}}({{args}}) ({{results}}) {
	{{firstLetter}}.mu.Lock()
	defer {{firstLetter}}.mu.Unlock()
	if {{firstLetter}}.alreadyCalled {
		return errors.New("rate limit exceeded")
	}
	{{firstLetter}}.alreadyCalled = true
	return {{firstLetter}}.{{fnName}}({{varArgs}})
}`)

	return text.ToDecl(t)
}
