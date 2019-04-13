package hilbridge

import (
	"bytes"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
)

// VariableExprFunc is a function type used to adapt HIL's idea of variables
// (an single string possibly containing dots and asterisks along with
// identifiers) to the HCL 2 model where dots and asterisks traverse through
// nested objects in the EvalContext.
//
// This is customizable because in HIL-based applications the interpretation
// of variable reference strings was delegated entirely to the calling
// application, and thus we cannot model all possible application logic via
// a standard implementation here. However, in practice applications tended
// to treat dot as an attribute traversal and attribute names consisting
// entirely of digits as list index lookups, which is implemented by the
// "default" lookup function VariableLookupExpr.
type VariableExprFunc func(name string, rng hcl.Range) hcl.Expression

// VariableLookupExpr transforms a string containing dot-separated
// attribute/index traversals into a HCL expression that reads a value
// from its given EvalContext.
//
// Dot-separated traversals was a common interpretation of HIL variable strings
// in existing applications, so this may be a suitable VariableExprFunc to use
// in such applications.
//
// This function may attempt to generate derived source ranges from the given
// range by following the usual assumptions that HCL 2 makes about characters.
// The results are undefined if the given range is not a reasonable range for
// the given name string, such as if the distance between the start and end
// are not consistent with the number of characters and bytes in the string.
//
// Note that this does _not_ handle Terraform-style "splat" references, because
// they did not generalize to all source expressions in Terraform versions
// prior to v0.12 where HIL was used. Terraform does not use hilbridge itself,
// but if you want to use hilbridge in a Terraform-compatible way you will need
// to implement a custom VariableExprFunc to recognize and handle splat
// references in a manner compatible with Terraform's old interpolator.
func VariableLookupExpr(name string, rng hcl.Range) hcl.Expression {
	traversal, diags := VariableTraversal(name, rng)
	if diags.HasErrors() {
		return &failExpr{
			diags: diags,
			rng:   rng,
		}
	}
	return variableLookupExpr{
		traversal: traversal,
	}
}

// VariableTraversal transforms a string containing dot-separated attribute/index
// traversals into a HCL traversal.
//
// This function produces an absolute traversal that has the same effect as the
// expression produced by VariableLookupExpr. See the documentation of that
// function for more details.
func VariableTraversal(name string, rng hcl.Range) (hcl.Traversal, hcl.Diagnostics) {
	sc := hcl.NewRangeScannerFragment([]byte(name), rng.Filename, rng.Start, scanAttrSteps)
	var ret hcl.Traversal
	for sc.Scan() {
		name := string(sc.Bytes())
		rng := sc.Range()
		if len(ret) == 0 {
			ret = append(ret, hcl.TraverseRoot{
				// First step must always be a proper variable name, since we
				// will look it up in the scope.
				Name:     name,
				SrcRange: rng,
			})
		} else {
			ret = append(ret, hcl.TraverseIndex{
				// We're cheating a bit here: the hcl.Index function attempts
				// automatic conversion of the key to an integer if traversing
				// through a list, so we can safely just always set a string
				// here and let it fail at traversal time if we try to go
				// through a list with a string that isn't just decimal digits.
				Key:      cty.StringVal(name),
				SrcRange: rng,
			})
		}
	}
	if len(ret) == 0 {
		return nil, hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Empty variable reference",
				Detail:   "There must be at least one character in a variable reference.",
				Subject:  &rng,
			},
		}
	}
	return ret, nil
}

var _ VariableExprFunc = VariableLookupExpr

type variableLookupExpr struct {
	traversal hcl.Traversal
}

func (e variableLookupExpr) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	return e.traversal.TraverseAbs(ctx)
}

func (e variableLookupExpr) Variables() []hcl.Traversal {
	return []hcl.Traversal{e.traversal}
}

func (e variableLookupExpr) StartRange() hcl.Range {
	return e.traversal.SourceRange()
}

func (e variableLookupExpr) Range() hcl.Range {
	return e.traversal.SourceRange()
}

type failExpr struct {
	diags hcl.Diagnostics
	rng   hcl.Range
}

func (e failExpr) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	return cty.DynamicVal, e.diags
}

func (e failExpr) Variables() []hcl.Traversal {
	return nil
}

func (e failExpr) StartRange() hcl.Range {
	return e.rng
}

func (e failExpr) Range() hcl.Range {
	return e.rng
}

func scanAttrSteps(data []byte, atEOF bool) (advance int, token []byte, err error) {
	idx := bytes.IndexByte(data, '.')
	if idx < 0 {
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	}
	return idx, data[:idx], nil
}
