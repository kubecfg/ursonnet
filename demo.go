package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/toolutils"
	"github.com/mkmik/ursonnet/internal/unparser"
)

const (
	ursonnetTraceTag = "uRsOnNeT"
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
	dump(a, 0)

	vm := jsonnet.MakeVM()

	log.Printf("Injecting traces")
	if err := walk(a); err != nil {
		return err
	}

	fmt.Println("After:")
	fmt.Println(unparse(a))
	dump(a, 0)

	res, err := vm.Evaluate(a)
	if err != nil {
		return err
	}
	fmt.Println(res)

	return nil
}

func unparse(a ast.Node) string {
	u := unparser.Unparser{}
	u.Unparse(a, false)
	return u.String()
}

func dump(a ast.Node, indent int) {
	log.Printf("%s%T, free vars: %v", strings.Repeat(" ", indent), a, a.FreeVariables())
	for _, c := range toolutils.Children(a) {
		dump(c, indent+2)
	}
}

func addFreeVariable(n ast.Identifier, a ast.Node) {
	vars := a.FreeVariables()
	for _, v := range vars {
		if v == n {
			return
		}
	}
	vars = append(vars, n)
	a.SetFreeVariables(vars)
}

// walk walks the AST depth first
func walk(a ast.Node) error {
	for _, c := range toolutils.Children(a) {
		if err := walk(c); err != nil {
			return err
		}
	}

	// percolate "std" free variable up the tree
	addFreeVariable("std", a)

	if false {
		log.Printf("walking: %T, free vars: %v", a, a.FreeVariables())
	}
	if o, ok := a.(*ast.DesugaredObject); ok {
		for i, field := range o.Fields {
			if false {
				log.Printf("desugared object field: %s", unparse(field.Name))
			}

			var tbase ast.NodeBase = o.NodeBase
			tbase.SetContext(field.Body.Context())
			tbase.SetFreeVariables(field.Body.FreeVariables())
			if loc := tbase.Loc(); loc != nil {
				*loc = *field.Body.Loc()
			}
			trace := ast.Apply{
				NodeBase: tbase,
				Target: &ast.Index{
					NodeBase: tbase,
					Target:   &ast.Var{Id: ast.Identifier("std")},
					Index:    &ast.LiteralString{NodeBase: tbase, Value: "trace"},
				},
				Arguments: ast.Arguments{
					Positional: []ast.CommaSeparatedExpr{
						{Expr: &ast.LiteralString{NodeBase: tbase, Value: ursonnetTraceTag}},
						{Expr: field.Body},
					},
				},
			}
			name := field.Name.(*ast.LiteralString).Value
			if true {
				log.Printf("TRACE into %v LOOKS LIKE: %s", name, unparse(&trace))
				dump(&trace, 4)
			}
			if _, isObj := field.Body.(*ast.DesugaredObject); !isObj {
				o.Fields[i].Body = &trace
			}
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
