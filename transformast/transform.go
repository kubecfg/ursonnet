package transformast

import (
	"fmt"

	"github.com/google/go-jsonnet/ast"
)

type NodeTransformer func(ast.Node) (ast.Node, error)

func Transform(node ast.Node, fun NodeTransformer) (res ast.Node, err error) {
	type transformError struct{ error }

	// only captures panics issued by `tr` and converts them into normal errors
	defer func() {
		if r := recover(); r != nil {
			if te, ok := r.(transformError); ok {
				err = te.error
			}
		}
	}()

	tr := func(n *ast.Node) {
		var err error
		*n, err = Transform(*n, fun)
		if err != nil {
			panic(transformError{err})
		}
	}

	switch node := (node).(type) {
	case *ast.Apply:
		tr(&node.Target)
	case *ast.ApplyBrace:
		tr(&node.Left)
		tr(&node.Right)
	case *ast.Array:
		for i := range node.Elements {
			tr(&node.Elements[i].Expr)
		}
	case *ast.ArrayComp:
		tr(&node.Body)
		tr(&node.Spec.Expr)
		for i := range node.Spec.Conditions {
			tr(&node.Spec.Conditions[i].Expr)
		}
	case *ast.Assert:
		tr(&node.Cond)
		tr(&node.Message)
		tr(&node.Rest)
	case *ast.Binary:
		tr(&node.Left)
		tr(&node.Right)
	case *ast.Index:
		tr(&node.Target)
		tr(&node.Index)
	case *ast.Local:
		for i := range node.Binds {
			tr(&node.Binds[i].Body)
		}
		tr(&node.Body)
	case *ast.DesugaredObject:
		for i := range node.Fields {
			tr(&node.Fields[i].Name)
			tr(&node.Fields[i].Body)
		}
	case *ast.Unary:
		tr(&node.Expr)
	case *ast.InSuper:
		tr(&node.Index)
	case *ast.Parens:
		tr(&node.Inner)
	case *ast.SuperIndex:
		tr(&node.Index)
	case *ast.Conditional:
		tr(&node.Cond)
		tr(&node.BranchTrue)
		tr(&node.BranchFalse)
	case *ast.Error:
		tr(&node.Expr)
	case *ast.Function:
		tr(&node.Body)
		for i := range node.Parameters {
			tr(&node.Parameters[i].DefaultArg)
		}
	case *ast.Slice:
		tr(&node.Target)
		tr(&node.BeginIndex)
		tr(&node.EndIndex)
		tr(&node.Step)
	case *ast.Object:
		for i := range node.Fields {
			tr(&node.Fields[i].Expr1)
			tr(&node.Fields[i].Expr2)
			tr(&node.Fields[i].Expr3)
		}
	case *ast.ObjectComp:
		for i := range node.Fields {
			tr(&node.Fields[i].Expr1)
			tr(&node.Fields[i].Expr2)
			tr(&node.Fields[i].Expr3)
		}
		tr(&node.Spec.Expr)
		for i := range node.Spec.Conditions {
			tr(&node.Spec.Conditions[i].Expr)
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
