package ursonnet

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/toolutils"
	"github.com/mkmik/ursonnet/internal/unparser"
	"github.com/mkmik/ursonnet/transformast"
)

const (
	ursonnetTraceTag = "uRsOnNeT"
)

type RootsOpt func(opts *rootsOptions)

type rootsOptions struct{ debug bool }

func Debug(v bool) RootsOpt {
	return func(opts *rootsOptions) {
		opts.debug = v
	}
}

// Roots evaluates an expression in the context of a jsonnet file identified by the filename import path,
// and returns a slice of "file:linenumber" location of potentially "interesting" places where an edit in the
// jsonnet source files will have an effect on the evaluation of expression (a "causal trace").
//
// Clobbers the trace output writer by calling `vm.SetTraceOut` without being able to save the previous value.
// The main reason why `vm` is passed here is to allow the caller to setup their own importer/import paths, ext vars etc.
func Roots(vm *jsonnet.VM, filename string, expr string, opts ...RootsOpt) ([]string, error) {
	var opt rootsOptions
	for _, o := range opts {
		o(&opt)
	}

	root, err := jsonnet.SnippetToAST(ursonnetTraceTag, fmt.Sprintf("((import %q)+{ __ursonnet_res_:: %s}).__ursonnet_res_", filename, expr))
	if err != nil {
		return nil, err
	}
	if opt.debug {
		fmt.Println("Before expansion:")
		fmt.Println(unparse(root))
	}

	root = expandImports(vm, root)
	root = expandImports(vm, root)
	// todo fix recursive imports

	if opt.debug {
		fmt.Println("After import expansion:")
		fmt.Println(unparse(root))
	}

	if err := injectTrace(root); err != nil {
		return nil, err
	}

	if opt.debug {
		fmt.Println("After inject trace:")
		fmt.Println(unparse(root))
	}

	var traceOut bytes.Buffer
	vm.SetTraceOut(&traceOut)

	evalResult, err := vm.Evaluate(root)
	if err != nil {
		return nil, err
	}
	if opt.debug {
		log.Printf("Res: %s", evalResult)
	}

	// the traceOut buffer will contain all user defined traces intermixed with the traces that we injected.
	// All trace lines will look like:
	//
	//    TRACE: <filename>:<linenumber> <message>
	//
	// Our traces will look like:
	//    TRACE: <filename>:<linenumber> {{ursonnetTraceTag}}
	//
	// The top-level expression that evaluates `expr` is also traced, but we don't want the user to see that trace.
	// It's easier to filter that line out here since for that line `<filename> == {ursonnetTraceTag}`

	ignoreLine := fmt.Sprintf("TRACE: %s:1 %s", ursonnetTraceTag, ursonnetTraceTag)

	var res []string
	scanner := bufio.NewScanner(&traceOut)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, ursonnetTraceTag) {
			if line == ignoreLine {
				continue
			}
			res = append(res, strings.TrimPrefix(strings.TrimSuffix(line, ursonnetTraceTag), "TRACE: "))
		}
	}

	return res, nil
}

func expandImports(vm *jsonnet.VM, a ast.Node) ast.Node {
	return transformast.Transform(a, func(node ast.Node) ast.Node {
		if node, ok := node.(*ast.Import); ok {
			a, _, err := vm.ImportAST(node.Loc().FileName, node.File.Value)
			if err != nil {
				panic(err) // TODO: convert to error
			}
			return a
		}
		return node
	})
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
	addFreeVariable("$std", a) // this is a special variable used when desugaring comprehensions

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
