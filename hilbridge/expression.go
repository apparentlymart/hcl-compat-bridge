package hilbridge

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hil"
	hilast "github.com/hashicorp/hil/ast"
	"github.com/zclconf/go-cty/cty"
)

// Parse parses the given string for HIL interpolation sequences and returns an
// HCL expression that will return the result of evaluating that string.
//
// The given filename and start position describe where this HIL expression
// was found. Any source ranges generated against the expression will be
// relative to this start position.
//
// If the returned diagnostics contains errors, the returned expression is
// invalid and cannot be used.
func Parse(v string, filename string, start hcl.Pos) (*Expression, hcl.Diagnostics) {
	node, err := hil.ParseWithPosition(v, posHCLtoHIL(filename, start))
	if err != nil {
		return nil, errorToDiagnostics(err, []byte(v), start)
	}
	e := &Expression{
		node:  node,
		src:   []byte(v),
		start: start,
	}

	node.Accept(fixUpASTVisitor(e))

	return e, nil
}

// Expression is an implementation of hcl.Expression that adapts the Expression
// API to the HIL parser and evaluator.
type Expression struct {
	node  hilast.Node
	src   []byte
	start hcl.Pos
}

func (e *Expression) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	result, err := hil.Eval(e.node, &hil.EvalConfig{
		GlobalScope: &hilast.BasicScope{
			// Because we've replaced the variable access and function call
			// nodes in the AST, we can construct this rather odd-shaped
			// VarMap and still get the expected result. This is how we can
			// smuggle our *hcl.EvalContext down into the HIL evaluation
			// codepath.
			VarMap: map[string]hilast.Variable{
				"ctx": {
					Value: ctx,
				},
			},
		},
	})
	if err != nil {
		return cty.DynamicVal, errorToDiagnostics(err, e.src, e.start)
	}

	// TODO
	return cty.DynamicVal, nil
}

func (e *Expression) Variables() []hcl.Traversal {
	return nil
}

func (e *Expression) Range() hcl.Range {
	return rangeHILtoHCL(e.node.Pos(), e.src, e.start)
}

func (e *Expression) StartRange() hcl.Range {
	return e.Range()
}
