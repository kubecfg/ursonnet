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
	"github.com/kubecfg/ursonnet/internal/unparser"
	"github.com/kubecfg/ursonnet/transformast"
)

const (
	ursonnetTraceTag = "uRsOnNeT"
)

// RootsOpt is an option for Roots
type RootsOpt func(opts *rootsOptions)

type rootsOptions struct{ debug bool }

// Debug sets whether Roots emits verbose debug logs.
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

	root, err = expandImports(vm, root, map[string]bool{})
	if err != nil {
		return nil, err
	}

	if opt.debug {
		fmt.Println("After import expansion:")
		fmt.Println(unparse(root))
	}

	if err := injectTrace(root, map[ast.Node]bool{}); err != nil {
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

	seen := map[string]bool{}

	var res []string
	scanner := bufio.NewScanner(&traceOut)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, ursonnetTraceTag) {
			if line == ignoreLine {
				continue
			}
			clean := strings.TrimPrefix(strings.TrimSuffix(line, ursonnetTraceTag), "TRACE: ")
			if !seen[clean] {
				res = append(res, clean)
				seen[clean] = true
			}
		}
	}
	reverse(res)

	return res, nil
}

func expandImports(vm *jsonnet.VM, a ast.Node, seen map[string]bool) (ast.Node, error) {
	return transformast.Transform(a, func(node ast.Node) (ast.Node, error) {
		if node, ok := node.(*ast.Import); ok {
			a, foundAt, err := vm.ImportAST(node.Loc().FileName, node.File.Value)
			if err != nil {
				return nil, err
			}
			if seen[foundAt] {
				return a, nil
			}
			seen[foundAt] = true
			return expandImports(vm, a, seen)
		}
		return node, nil
	})
}

// injectTrace walks the AST depth first
func injectTrace(a ast.Node, seen map[ast.Node]bool) error {
	if seen[a] {
		return nil
	}
	seen[a] = true

	for _, c := range toolutils.Children(a) {
		if err := injectTrace(c, seen); err != nil {
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
				// I was tempted to use field.Body.Loc() but it turns out that's not
				// initialized in some desugarings like `f(arg): body` -> `f: function(arg) body`.
				// OTOH, the field location itself is always available
				*loc = field.LocRange
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

func reverse[T any](s []T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func dump(a ast.Node, indent int) {
	log.Printf("%s%T, free vars: %v", strings.Repeat(" ", indent), a, a.FreeVariables())
	for _, c := range toolutils.Children(a) {
		dump(c, indent+2)
	}
}
