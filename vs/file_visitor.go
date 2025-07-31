package vs

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

type FileVisitor struct {
	FSet   *token.FileSet
	File   *ast.File
	Modify bool
}

func (v *FileVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return v
	}
	switch n := node.(type) {
	case *ast.FuncDecl:
		if strings.HasPrefix(n.Name.Name, "Test") || n.Body == nil {
			return v
		}
		v.addTimecostToFunc(n)
		v.Modify = true
	}
	return v
}

func (v *FileVisitor) addTimecostToFunc(node *ast.FuncDecl) {
	v.AddRuntimeIfNeed()
}

func (v *FileVisitor) AddRuntimeIfNeed() {
	astutil.AddImport(v.FSet, v.File, "time")
	astutil.AddImport(v.FSet, v.File, "fmt")
}
