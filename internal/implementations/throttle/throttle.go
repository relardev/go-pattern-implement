package throttle

import (
	"component-generator/internal/code"
	"component-generator/internal/fstr"
	"component-generator/internal/naming"
	"component-generator/internal/text"
	"fmt"
	"go/ast"
	"unicode"
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
		return "throttle"
	}

	return "throttle-error"
}

func (i *Implementator) Description() string {
	if i.mode == ModeNoError {
		return "Process at most n requests per second, on throttled call return no error"
	}

	return "Process at most n requests per second, on throttled call return an error"
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
				if i.mode == ModeNoError {
					validateNoError(methodDef)
				}

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
	results := code.AddPackageNameToResults(field.Type.(*ast.FuncType).Results, i.packageName)

	zeroReturns := i.getThrottledReturn(results)

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
	}, `
func ({{firstLetter}} *Throttle) {{fnName}}({{args}}) ({{results}}) {
	{{firstLetter}}.mu.Lock()
	if {{firstLetter}}.alreadyCalled {
		{{firstLetter}}.mu.Unlock()
		return {{zeroReturns}}
	}
	{{firstLetter}}.alreadyCalled = true
	{{firstLetter}}.mu.Unlock()
	{{returnPart}}
}`)

	return text.ToDecl(t)
}

func (i *Implementator) getThrottledReturn(results *ast.FieldList) []ast.Expr {
	if results == nil {
		return nil
	}

	if i.mode == ModeNoError {
		// only possible return can be an error
		return []ast.Expr{
			ast.NewIdent("nil"),
		}
	}

	zeroReturns := []ast.Expr{}

	for _, r := range results.List {
		zeroReturns = append(zeroReturns, code.ZeroValue(r.Type))
	}

	returnsError, errorPos := code.DoesFieldListReturnError(results)
	if returnsError {
		zeroReturns[errorPos] = text.ToExpr("errors.New(\"rate limit exceeded\")")
	}

	return zeroReturns
}

func validateNoError(field *ast.Field) {
	returns := field.Type.(*ast.FuncType).Results
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
