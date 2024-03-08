package code

import (
	"go/ast"
	"go/token"
	"unicode"
)

type StructField struct {
	Name string

	// We need either TypeStr or TypeSpec
	TypeStr  string
	TypeSpec ast.Expr
}

func Struct(name string, fields ...StructField) ast.Decl {
	specs := []*ast.Field{}
	for _, field := range fields {
		var t ast.Expr
		if field.TypeSpec != nil {
			t = field.TypeSpec
		} else {
			t = ast.NewIdent(field.TypeStr)
		}

		specs = append(specs, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(field.Name)},
			Type:  t,
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
		Name:    string(lowerFirstLetter),
		TypeStr: packageName + "." + name,
	}
}
