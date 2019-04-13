package hilbridge

import (
	"fmt"
	"github.com/hashicorp/hcl2/hcl"
	hilparser "github.com/hashicorp/hil/parser"
)

func errorToDiagnostics(err error, src []byte, srcPos hcl.Pos) hcl.Diagnostics {
	if err == nil {
		return nil
	}
	switch err := err.(type) {
	case *hilparser.ParseError:
		pos := err.Pos
		rng := rangeHILtoHCL(pos, src, srcPos)
		return hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Error during parsing",
				Detail:   fmt.Sprintf("Invalid syntax: %s.", err.Message),
				Subject:  &rng,
			},
		}
	default:
		return hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid interpolation",
				Detail:   fmt.Sprintf("Failed: %s.", err.Error()),
			},
		}
	}
}
