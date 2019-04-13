package hilbridge

import (
	"github.com/hashicorp/hil"
	hilast "github.com/hashicorp/hil/ast"
)

func fixUpASTVisitor(expr *Expression) hilast.Visitor {
	return func(node hilast.Node) hilast.Node {
		switch node := node.(type) {
		case *hilast.VariableAccess:
			return &variableAccessNode{
				Name: node.Name,
				Posx: node.Posx,
				Expr: expr,
			}
		case *hilast.Call:
			return &functionCallNode{
				Func: node.Func,
				Args: node.Args,
				Posx: node.Posx,
				Expr: expr,
			}
		default:
			return node
		}
	}
}

// variableAccessNode is a replacement for HIL's own VariableAccess node which
// looks up variables in a HCL EvalContext instead of directly in the HIL
// scope.
type variableAccessNode struct {
	Name string
	Posx hilast.Pos
	Expr *Expression
}

var _ hilast.Node = (*variableAccessNode)(nil)
var _ hil.EvalNode = (*variableAccessNode)(nil)

func (n *variableAccessNode) Accept(v hilast.Visitor) hilast.Node {
	return v(n)
}

func (n *variableAccessNode) Pos() hilast.Pos {
	return n.Posx
}

func (n *variableAccessNode) Type(s hilast.Scope) (hilast.Type, error) {
	return hilast.TypeUnknown, nil
}

func (n *variableAccessNode) Eval(s hilast.Scope, _ *hilast.Stack) (interface{}, hilast.Type, error) {
	return nil, hilast.TypeUnknown, nil
}

// functionCallNode is a replacement for HIL's own Call node which
// calls functions in a HCL EvalContext instead of directly in the HIL scope.
type functionCallNode struct {
	Func string
	Args []hilast.Node
	Posx hilast.Pos
	Expr *Expression
}

var _ hilast.Node = (*functionCallNode)(nil)
var _ hil.EvalNode = (*functionCallNode)(nil)

func (n *functionCallNode) Accept(v hilast.Visitor) hilast.Node {
	for i, a := range n.Args {
		n.Args[i] = a.Accept(v)
	}
	return v(n)
}

func (n *functionCallNode) Pos() hilast.Pos {
	return n.Posx
}

func (n *functionCallNode) Type(s hilast.Scope) (hilast.Type, error) {
	return hilast.TypeUnknown, nil
}

func (n *functionCallNode) Eval(s hilast.Scope, _ *hilast.Stack) (interface{}, hilast.Type, error) {
	return nil, hilast.TypeUnknown, nil
}
