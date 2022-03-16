package transformast

import "github.com/google/go-jsonnet/ast"

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
	case *ast.Conditional:
		// TODO: fill in all other stuff
	case *ast.Dollar:

	case *ast.Error:

	case *ast.Function:

	case *ast.Import:

	case *ast.ImportStr:

	case *ast.Index:

	case *ast.InSuper:

	case *ast.LiteralBoolean:

	case *ast.LiteralNull:

	case *ast.LiteralNumber:

	case *ast.LiteralString:

	case *ast.Local:
		for i := range node.Binds {
			expand(&node.Binds[i].Body)
		}
		expand(&node.Body)
	case *ast.Object:

	case *ast.ObjectComp:

	case *ast.Parens:

	case *ast.Self:

	case *ast.Slice:

	case *ast.SuperIndex:

	case *ast.Unary:

	case *ast.Var:

	}
	return fun(node)
}
