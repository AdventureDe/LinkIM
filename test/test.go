package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

func main() {
	// 解析 Go 文件
	filePath := "../user/handler/handler.go" // 替换为你的文件路径
	fs := token.NewFileSet()
	node, err := parser.ParseFile(fs, filePath, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	// 创建 txt 文件
	outFile, err := os.Create("func_names.txt")
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	// 遍历 AST，提取函数名
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			// 输出到终端
			fmt.Println(fn.Name.Name)
			// 写入 txt 文件
			_, _ = outFile.WriteString(fn.Name.Name + "\n")
		}
	}

	fmt.Println("函数名已写入 func_names.txt")
}
