package main

import (
	"bytes"
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
	Path      string `arg:"" type:"path"`
	FieldPath string `arg:"" default:"$" help:"jsonnet field path, example, $.a.b"`
}

func (cmd *CLI) Run(cli *Context) error {
	debug := false

	b, err := ioutil.ReadFile(cmd.Path)
	if err != nil {
		return err
	}
	a, err := jsonnet.SnippetToAST(cmd.Path, string(b))
	if err != nil {
		return err
	}

	if debug {
		fmt.Println("Before:")
		fmt.Println(unparse(a))
		dump(a, 0)
	}

	vm := jsonnet.MakeVM()

	if err := injectTrace(a); err != nil {
		return err
	}

	if debug {
		fmt.Println("After:")
		fmt.Println(unparse(a))
		dump(a, 0)
	}

	root, err := jsonnet.SnippetToAST("", fmt.Sprintf("(function(x)(x+{ res:: %s}).res)(null)", cmd.FieldPath))
	if err != nil {
		return err
	}
	root.(*ast.Apply).Arguments.Positional[0].Expr = a
	addFreeVariable("std", root)

	var traceOut bytes.Buffer
	vm.SetTraceOut(&traceOut)

	res, err := vm.Evaluate(root)
	if err != nil {
		return err
	}
	fmt.Println(res)

	fmt.Println(traceOut.String())

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

// injectTrace walks the AST depth first
func injectTrace(a ast.Node) error {
	for _, c := range toolutils.Children(a) {
		if err := injectTrace(c); err != nil {
			return err
		}
	}

	// percolate "std" free variable up the tree
	addFreeVariable("std", a)

	if o, ok := a.(*ast.DesugaredObject); ok {
		for i, field := range o.Fields {

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
