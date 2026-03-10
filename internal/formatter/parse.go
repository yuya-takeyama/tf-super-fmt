package formatter

import (
	"bytes"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// Parse parses an HCL source file and builds a FileModel.
func Parse(path string, src []byte) (*FileModel, error) {
	eol := DetectEOL(src)
	normalized := NormalizeToLF(src)

	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(normalized, path)
	if diags.HasErrors() {
		return nil, diags
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "unexpected body type",
		}
	}

	lines := splitLines(normalized)
	classifiedLines := classifyLines(lines)

	bodyModel := buildBodyModel(body, classifiedLines, 0, true)

	return &FileModel{
		Path:  path,
		Src:   normalized,
		EOL:   eol,
		Lines: classifiedLines,
		Body:  bodyModel,
	}, nil
}

// splitLines splits source into lines (without EOL characters).
func splitLines(src []byte) [][]byte {
	if len(src) == 0 {
		return nil
	}
	raw := bytes.Split(src, []byte("\n"))
	// Remove trailing empty element if source ends with newline
	if len(raw) > 0 && len(raw[len(raw)-1]) == 0 {
		raw = raw[:len(raw)-1]
	}
	return raw
}

// classifyLines converts raw line bytes into Line structs with kind classification.
func classifyLines(rawLines [][]byte) []Line {
	lines := make([]Line, len(rawLines))
	depth := 0
	for i, content := range rawLines {
		trimmed := bytes.TrimLeft(content, " \t")
		kind := classifyLine(trimmed)
		lines[i] = Line{
			Number:  i + 1,
			Content: content,
			Kind:    kind,
			Depth:   depth,
		}
		// Update depth for next line
		if kind == LineBlockOpen {
			depth++
		} else if kind == LineBlockClose {
			if depth > 0 {
				depth--
			}
			// Fix the depth of the closing brace itself
			lines[i].Depth = depth
		} else if kind == LineOneLineBlock {
			// no change
		}
	}
	return lines
}

// classifyLine determines the kind of a line based on its trimmed content.
func classifyLine(trimmed []byte) LineKind {
	if len(trimmed) == 0 {
		return LineBlank
	}
	s := string(trimmed)
	if strings.HasPrefix(s, "#") || strings.HasPrefix(s, "//") || strings.HasPrefix(s, "/*") {
		return LineCommentOnly
	}
	if s == "}" || s == "})" {
		return LineBlockClose
	}
	// Check for one-line block: ends with } and has { somewhere
	if strings.Contains(s, "{") && strings.HasSuffix(strings.TrimRight(s, " \t"), "}") {
		return LineOneLineBlock
	}
	// Check for block open: ends with {
	if strings.HasSuffix(strings.TrimRight(s, " \t"), "{") {
		return LineBlockOpen
	}
	// Check for attribute: has = not inside a string (simple heuristic)
	if hasAttributeEquals(s) {
		return LineAttributeHeader
	}
	return LineContinuation
}

// hasAttributeEquals checks if a line has an = sign that likely denotes an attribute.
func hasAttributeEquals(s string) bool {
	// Find = that is not == and not preceded by !, <, >, =
	for i, ch := range s {
		if ch == '=' {
			if i+1 < len(s) && s[i+1] == '=' {
				continue // ==
			}
			if i > 0 {
				prev := s[i-1]
				if prev == '!' || prev == '<' || prev == '>' || prev == '=' {
					continue
				}
			}
			return true
		}
	}
	return false
}

// rawItem is an intermediate representation used during parsing.
type rawItem struct {
	kind      ItemKind
	startLine int
	endLine   int
	name      string
	attr      *hclsyntax.Attribute
	block     *hclsyntax.Block
}

// buildBodyModel constructs a BodyModel from an hclsyntax.Body.
func buildBodyModel(body *hclsyntax.Body, lines []Line, depth int, isTopLevel bool) *BodyModel {
	if body == nil {
		return nil
	}

	startLine := body.SrcRange.Start.Line
	endLine := body.SrcRange.End.Line

	var items []rawItem

	for name, attr := range body.Attributes {
		items = append(items, rawItem{
			kind:      ItemAttribute,
			startLine: attr.SrcRange.Start.Line,
			endLine:   attr.SrcRange.End.Line,
			name:      name,
			attr:      attr,
		})
	}

	for _, block := range body.Blocks {
		items = append(items, rawItem{
			kind:      ItemBlock,
			startLine: block.OpenBraceRange.Start.Line,
			endLine:   block.CloseBraceRange.End.Line,
			name:      block.Type,
			block:     block,
		})
	}

	// Sort by start line
	sortItems(items)

	// Build regions by associating leading comments and blank lines
	var regions []Region
	prevEndLine := startLine // track where previous item ended

	for _, item := range items {
		// Count blank lines before this item (between prevEndLine+1 and item.startLine-1)
		blanks := 0
		commentLines := []int{}

		// Scan lines between prevEndLine and item.startLine
		for lineIdx := prevEndLine; lineIdx < item.startLine-1 && lineIdx < len(lines); lineIdx++ {
			l := lines[lineIdx]
			if l.Kind == LineBlank {
				blanks++
			} else if l.Kind == LineCommentOnly {
				// Leading comment for this item - reset blank count
				commentLines = append(commentLines, l.Number)
				blanks = 0
			}
		}

		im := ItemModel{
			StartLine: item.startLine,
			EndLine:   item.endLine,
			Name:      item.name,
		}

		if item.kind == ItemAttribute {
			im.Kind = ItemAttribute
			im.IsMultiLineValue = item.startLine != item.endLine
		} else {
			im.Kind = ItemBlock
			im.IsOneLine = item.block.OpenBraceRange.Start.Line == item.block.CloseBraceRange.End.Line
			if !im.IsOneLine && item.block.Body != nil {
				im.ChildBody = buildBodyModel(item.block.Body, lines, depth+1, false)
			}
		}

		regions = append(regions, Region{
			LeadingCommentLines: commentLines,
			Item:                im,
			BlanksBefore:        blanks,
		})

		prevEndLine = item.endLine
	}

	return &BodyModel{
		Depth:      depth,
		StartLine:  startLine,
		EndLine:    endLine,
		IsTopLevel: isTopLevel,
		Regions:    regions,
	}
}

// sortItems sorts rawItems by startLine.
func sortItems(items []rawItem) {
	// Simple insertion sort
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].startLine < items[j-1].startLine; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}
