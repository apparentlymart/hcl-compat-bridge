package hilbridge

import (
	"bufio"
	"github.com/apparentlymart/go-textseg/textseg"
	"github.com/hashicorp/hcl2/hcl"
	hilast "github.com/hashicorp/hil/ast"
)

func posHCLtoHIL(filename string, pos hcl.Pos) hilast.Pos {
	return hilast.Pos{
		Filename: filename,
		Line:     pos.Line,
		Column:   pos.Column,
	}
}

func posHILtoHCL(pos hilast.Pos, src []byte, srcPos hcl.Pos) hcl.Pos {
	rng := rangeHILtoHCL(pos, src, srcPos)
	return rng.Start
}

func rangeHILtoHCL(pos hilast.Pos, src []byte, srcPos hcl.Pos) hcl.Range {
	// HIL positions are a subset of HCL positions in that they only track
	// line/column, not byte offset. Therefore we need to use the given
	// source code to try to reverse-engineer an accurate byte offset.
	// To do that, we'll first find the line of the input that contains the
	// position (because lines are equivalent between HIL and HCL) and then
	// count forward from that position to find our final byte offset.
	sc := hcl.NewRangeScannerFragment(src, pos.Filename, srcPos, bufio.ScanLines)
	for sc.Scan() {
		lineRng := sc.Range()
		if lineRng.Start.Line != pos.Line {
			continue
		}
		if lineRng.Start.Line > pos.Line {
			// Seems like we missed it then, so we might as well stop here.
			break
		}

		// HIL "columns" are counted in bytes, so the Column number from the
		// given position actually tells us how many _bytes_ after the start
		// we are, and we'll need to do some more work below to count the
		// characters in an HCL-like way to determine the column.
		byteOfs := pos.Column - 1
		if byteOfs < 0 {
			break // Invalid position given, so we can't help.
		}

		lineSrc := sc.Bytes()
		if byteOfs > len(lineSrc) {
			break // Somehow our position is off the end of the line, so we'll give up.
		}
		cols, err := textseg.TokenCount(lineSrc[:byteOfs], textseg.ScanGraphemeClusters)
		if err != nil {
			break // should never happen because ScanGraphemeClusters never returns errors
		}

		// Because HIL only tracks the start of things, we'll just let the range
		// run to the end of the line, so at least a diagnosting subject marker
		// will _start_ in the right place and have a non-zero length.
		return hcl.Range{
			Filename: pos.Filename,
			Start: hcl.Pos{
				Line:   lineRng.Start.Line,
				Column: lineRng.Start.Column + cols,
				Byte:   lineRng.Start.Byte + byteOfs,
			},
			End: lineRng.End,
		}
	}

	// If the above all failed then it's likely that we've been given inconsistent
	// information by the caller, so we'll just return a garbage placeholder.
	return hcl.Range{
		Filename: pos.Filename,
		Start:    hcl.Pos{},
		End:      hcl.Pos{},
	}
}
