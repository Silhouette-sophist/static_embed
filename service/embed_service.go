package service

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/ast/astutil"
)

const runtimeDir = "_runtime"

//go:embed _runtime
var _runtime embed.FS

func TraceAllRepoFunc(ctx context.Context, repoPath string) {
	info, err := os.Stat(repoPath)
	if err != nil {
		fmt.Printf("无效的路径: %v\n", err)
		os.Exit(1)
	}

	os.CopyFS(repoPath, _runtime)
	if info.IsDir() {
		// 处理目录
		if err := processDirectory(repoPath); err != nil {
			fmt.Printf("处理目录失败: %v\n", err)
		}
	} else {
		// 处理单个文件
		if err := addFunctionTiming(repoPath); err != nil {
			fmt.Printf("处理文件失败: %v\n", err)
		}
	}
}

// 为所有函数添加耗时统计代码
func addFunctionTiming(filePath string) error {
	fset := token.NewFileSet()

	// 解析源文件
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("解析文件失败: %v", err)
	}

	// 确保导入了必要的包
	astutil.AddImport(fset, file, "time")
	astutil.AddImport(fset, file, "fmt")

	// 使用astutil.Apply遍历并修改AST
	astutil.Apply(file,
		func(cursor *astutil.Cursor) bool {
			// 查找函数声明节点
			funcDecl, ok := cursor.Node().(*ast.FuncDecl)
			if !ok {
				return true // 不是函数声明，继续遍历
			}

			// 跳过测试函数
			if funcDecl.Name.Name == "Test" ||
				(len(funcDecl.Name.Name) > 4 && funcDecl.Name.Name[:4] == "Test") {
				return true
			}

			// 获取函数名（对于方法，包含接收者信息）
			funcName := getFunctionName(funcDecl)

			// 创建计时变量声明: start := time.Now()
			startStmt := &ast.AssignStmt{
				Lhs: []ast.Expr{
					ast.NewIdent("__start"),
				},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{
					&ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   ast.NewIdent("time"),
							Sel: ast.NewIdent("Now"),
						},
					},
				},
			}

			// 创建defer语句: defer func() { ... }()
			deferStmt := &ast.DeferStmt{
				Call: &ast.CallExpr{
					Fun: &ast.FuncLit{
						Type: &ast.FuncType{
							Params: &ast.FieldList{},
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								&ast.ExprStmt{
									X: &ast.CallExpr{
										Fun: &ast.SelectorExpr{
											X:   ast.NewIdent("fmt"),
											Sel: ast.NewIdent("Printf"),
										},
										Args: []ast.Expr{
											&ast.BasicLit{
												Kind:  token.STRING,
												Value: fmt.Sprintf("\"函数 %s 耗时: %%v\\n\"", funcName),
											},
											&ast.CallExpr{
												Fun: &ast.SelectorExpr{
													X:   ast.NewIdent("time"),
													Sel: ast.NewIdent("Since"),
												},
												Args: []ast.Expr{
													ast.NewIdent("__start"),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			// 在函数体开头插入新语句
			if funcDecl.Body != nil && len(funcDecl.Body.List) > 0 {
				// 在现有语句前插入
				funcDecl.Body.List = append(
					[]ast.Stmt{startStmt, deferStmt},
					funcDecl.Body.List...,
				)
			} else if funcDecl.Body != nil {
				// 函数体为空时直接添加
				funcDecl.Body.List = []ast.Stmt{startStmt, deferStmt}
			}

			fmt.Printf("已为函数 %s 添加耗时统计\n", funcName)
			return true
		},
		// post回调不需要处理
		func(cursor *astutil.Cursor) bool {
			return true
		},
	)

	// 生成修改后的代码
	var buf bytes.Buffer
	printerConfig := &printer.Config{
		Mode:     printer.UseSpaces | printer.TabIndent,
		Tabwidth: 4,
	}
	if err := printerConfig.Fprint(&buf, fset, file); err != nil {
		return fmt.Errorf("生成代码失败: %v", err)
	}

	// 写回文件
	return os.WriteFile(filePath, buf.Bytes(), 0644)
}

// 获取函数名（包含方法的接收者信息）
func getFunctionName(funcDecl *ast.FuncDecl) string {
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		// 处理方法（带接收者）
		recv := funcDecl.Recv.List[0]
		var recvStr string
		if ident, ok := recv.Type.(*ast.Ident); ok {
			recvStr = ident.Name
		} else if starExpr, ok := recv.Type.(*ast.StarExpr); ok {
			if ident, ok := starExpr.X.(*ast.Ident); ok {
				recvStr = "*" + ident.Name
			} else {
				recvStr = "*unknown"
			}
		} else {
			recvStr = "unknown"
		}
		return fmt.Sprintf("%s.%s", recvStr, funcDecl.Name.Name)
	}
	// 普通函数
	return funcDecl.Name.Name
}

// 批量处理目录下的所有Go文件
func processDirectory(dirPath string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理Go文件，跳过测试文件和目录
		if info.IsDir() || !(filepath.Ext(path) == ".go") ||
			filepath.Base(path) == "function_timing.go" { // 跳过自身
			return nil
		}

		fmt.Printf("处理文件: %s\n", path)
		return addFunctionTiming(path)
	})
}
