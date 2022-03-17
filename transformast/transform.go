package transformast

import (
	"fmt"

	"github.com/google/go-jsonnet/ast"
)

type NodeTransformer func(ast.Node) ast.Node

func Transform(node ast.Node, fun NodeTransformer) ast.Node {
	expand := func(n *ast.Node) {
		*n = Transform(*n, fun)
	}
	expandMany := func(ns []ast.CommaSeparatedExpr) {
		for i := range ns {
			ns[i].Expr = Transform(ns[i].Expr, fun)
		}
	}

	switch node := (node).(type) {
	case *ast.Apply:
		expand(&node.Target)
	case *ast.ApplyBrace:
		expand(&node.Left)
		expand(&node.Right)
	case *ast.Array:
		expandMany(node.Elements)
	case *ast.ArrayComp:
		expand(&node.Body)
		expand(&node.Spec.Expr)
		for i := range node.Spec.Conditions {
			expand(&node.Spec.Conditions[i].Expr)
		}
	case *ast.Assert:
		expand(&node.Cond)
		expand(&node.Message)
		expand(&node.Rest)
	case *ast.Binary:
		expand(&node.Left)
		expand(&node.Right)
	case *ast.Index:
		expand(&node.Target)
		expand(&node.Index)
	case *ast.Local:
		for i := range node.Binds {
			expand(&node.Binds[i].Body)
		}
		expand(&node.Body)
	case *ast.DesugaredObject:
		for i := range node.Fields {
			expand(&node.Fields[i].Name)
			expand(&node.Fields[i].Body)
		}
	case *ast.Unary:
		expand(&node.Expr)
	case *ast.InSuper:
		expand(&node.Index)
	case *ast.Parens:
		expand(&node.Inner)
	case *ast.SuperIndex:
		expand(&node.Index)
	case *ast.Conditional:
		expand(&node.Cond)
		expand(&node.BranchTrue)
		expand(&node.BranchFalse)
	case *ast.Error:
		expand(&node.Expr)
	case *ast.Function:
		expand(&node.Body)
		for i := range node.Parameters {
			expand(&node.Parameters[i].DefaultArg)
		}
	case *ast.Slice:
		expand(&node.Target)
		expand(&node.BeginIndex)
		expand(&node.EndIndex)
		expand(&node.Step)
	case *ast.Object:
		for i := range node.Fields {
			expand(&node.Fields[i].Expr1)
			expand(&node.Fields[i].Expr2)
			expand(&node.Fields[i].Expr3)
		}
	case *ast.ObjectComp:
		for i := range node.Fields {
			expand(&node.Fields[i].Expr1)
			expand(&node.Fields[i].Expr2)
			expand(&node.Fields[i].Expr3)
		}
		expand(&node.Spec.Expr)
		for i := range node.Spec.Conditions {
			expand(&node.Spec.Conditions[i].Expr)
		}
	case *ast.Import, *ast.ImportStr, *ast.Var:
		// only literal children
	case *ast.LiteralBoolean, *ast.LiteralNull, *ast.LiteralNumber, *ast.LiteralString:
		// only literal children
	case *ast.Self, *ast.Dollar:
		// no children
	default:
		panic(fmt.Sprintf("unhandled type %T", node))
	}
	return fun(node)
}
