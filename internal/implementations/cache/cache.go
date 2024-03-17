package cache

import "go/ast"

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
	return false, decls
}
