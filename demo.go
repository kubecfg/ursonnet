package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/alecthomas/kong"
	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/toolutils"
	"github.com/mkmik/ursonnet/internal/unparser"
)

type Context struct {
	*CLI
}

type CLI struct {
	Path string `arg:"" type:"path"`
}

func (cmd *CLI) Run(cli *Context) error {
	b, err := ioutil.ReadFile(cmd.Path)
	if err != nil {
		return err
	}
	a, err := jsonnet.SnippetToAST(cmd.Path, string(b))
	if err != nil {
		return err
	}

	fmt.Println("Before:")
	fmt.Println(unparse(a))

	if err := walk(a); err != nil {
		return err
	}

	fmt.Println("After:")
	fmt.Println(unparse(a))

	return nil
}

func unparse(a ast.Node) string {
	u := unparser.Unparser{}
	u.Unparse(a, false)
	return u.String()
}

// walk walks the AST depth first
func walk(a ast.Node) error {
	for _, c := range toolutils.Children(a) {
		if err := walk(c); err != nil {
			return err
		}
	}

	if false {
		log.Printf("walking: %s", unparse(a))
	}
	if o, ok := a.(*ast.DesugaredObject); ok {
		for i, field := range o.Fields {
			if false {
				log.Printf("desugared object field: %s", unparse(field.Name))
			}
			trace := ast.Apply{
				NodeBase: o.NodeBase,
				Target: &ast.Index{
					Target: &ast.Var{Id: ast.Identifier("std")},
					Id:     (func(a ast.Identifier) *ast.Identifier { return &a })(ast.Identifier("trace")),
				},
				Arguments: ast.Arguments{
					Positional: []ast.CommaSeparatedExpr{
						{Expr: &ast.LiteralString{}},
						{Expr: field.Body},
					},
				},
			}
			if false {
				log.Printf("TRACE LOOKS LIKE: %s", unparse(&trace))
			}
			o.Fields[i].Body = &trace
		}
	}

	return nil
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli)
	err := ctx.Run(&Context{CLI: &cli})
	ctx.FatalIfErrorf(err)
}
