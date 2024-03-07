package code

import (
	"go/ast"
	"go/token"
	"unicode"
)

type StructField struct {
	Name string
	Type string
}

func Struct(name string, fields ...StructField) ast.Decl {
	specs := []*ast.Field{}
	for _, field := range fields {
		specs = append(specs, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(field.Name)},
			Type:  ast.NewIdent(field.Type),
		})
	}
	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent(name),
				Type: &ast.StructType{
					Fields: &ast.FieldList{
						List: specs,
					},
				},
			},
		},
	}
}

func FieldFromTypeSpec(typeSpec *ast.TypeSpec, packageName string) StructField {
	name := typeSpec.Name.Name
	lowerFirstLetter := unicode.ToLower(rune(name[0]))
	return StructField{
		Name: string(lowerFirstLetter),
		Type: packageName + "." + name,
	}
}
