package formatter

import (
	"bytes"
	"strings"
)

// Rewrite takes a FileModel and produces formatted source bytes (LF line endings).
func Rewrite(fm *FileModel) []byte {
	var buf bytes.Buffer
	writeBody(&buf, fm.Body, fm.Lines)
	// Ensure exactly 1 newline at EOF
	result := buf.Bytes()
	result = bytes.TrimRight(result, "\n")
	result = append(result, '\n')
	return result
}

// writeBody writes the formatted content of a body.
func writeBody(buf *bytes.Buffer, body *BodyModel, lines []Line) {
	if body == nil || len(body.Regions) == 0 {
		return
	}

	for i, region := range body.Regions {
		// Determine blank lines to emit before this region
		blanksNeeded := blanksBeforeRegion(body, i)

		if i > 0 {
			for b := 0; b < blanksNeeded; b++ {
				buf.WriteByte('\n')
			}
		}

		// Write leading comment lines
		for _, lineNum := range region.LeadingCommentLines {
			lineIdx := lineNum - 1
			if lineIdx >= 0 && lineIdx < len(lines) {
				indent := strings.Repeat("  ", body.Depth)
				content := strings.TrimLeft(string(lines[lineIdx].Content), " \t")
				content = strings.TrimRight(content, " \t")
				buf.WriteString(indent)
				buf.WriteString(content)
				buf.WriteByte('\n')
			}
		}

		// Write item lines
		writeItem(buf, body, region.Item, lines)
	}
}

// blanksBeforeRegion determines how many blank lines to emit before region[i].
func blanksBeforeRegion(body *BodyModel, i int) int {
	if i == 0 {
		return 0
	}
	prev := body.Regions[i-1]
	curr := body.Regions[i]

	prevKind := prev.Item.Kind
	currKind := curr.Item.Kind

	if body.IsTopLevel {
		// Always exactly 1 blank line between top-level blocks
		return 1
	}

	// Nested body
	// Attribute ↔ Block boundary: always 1 blank line
	if prevKind != currKind {
		return 1
	}

	// Both attributes: preserve 0 or 1, compress 2+ to 1
	if prevKind == ItemAttribute && currKind == ItemAttribute {
		if curr.BlanksBefore > 0 {
			return 1
		}
		return 0
	}

	// Both blocks: preserve 0 or 1, compress 2+ to 1
	if curr.BlanksBefore > 0 {
		return 1
	}
	return 0
}

// writeItem writes a single item (attribute or block) with proper indentation.
func writeItem(buf *bytes.Buffer, body *BodyModel, item ItemModel, lines []Line) {
	indent := strings.Repeat("  ", body.Depth)

	if item.Kind == ItemAttribute {
		// Write all lines of this attribute
		for lineNum := item.StartLine; lineNum <= item.EndLine; lineNum++ {
			lineIdx := lineNum - 1
			if lineIdx < 0 || lineIdx >= len(lines) {
				continue
			}
			content := strings.TrimLeft(string(lines[lineIdx].Content), " \t")
			content = strings.TrimRight(content, " \t")
			if lineNum == item.StartLine {
				buf.WriteString(indent)
			} else {
				// Continuation lines: preserve relative indentation
				// Use body.Depth+1 as base for continuation
				buf.WriteString(indent)
				buf.WriteString("  ")
			}
			buf.WriteString(content)
			buf.WriteByte('\n')
		}
	} else {
		// Block
		if item.IsOneLine {
			lineIdx := item.StartLine - 1
			if lineIdx >= 0 && lineIdx < len(lines) {
				content := strings.TrimLeft(string(lines[lineIdx].Content), " \t")
				content = strings.TrimRight(content, " \t")
				buf.WriteString(indent)
				buf.WriteString(content)
				buf.WriteByte('\n')
			}
		} else {
			// Write opening line
			openIdx := item.StartLine - 1
			if openIdx >= 0 && openIdx < len(lines) {
				content := strings.TrimLeft(string(lines[openIdx].Content), " \t")
				content = strings.TrimRight(content, " \t")
				buf.WriteString(indent)
				buf.WriteString(content)
				buf.WriteByte('\n')
			}
			// Write child body
			if item.ChildBody != nil {
				writeBody(buf, item.ChildBody, lines)
			}
			// Write closing brace
			closeIdx := item.EndLine - 1
			if closeIdx >= 0 && closeIdx < len(lines) {
				buf.WriteString(indent)
				buf.WriteString("}")
				buf.WriteByte('\n')
			}
		}
	}
}

// AlignEquals reformats attribute lines in the buffer to align = signs.
// This is a post-processing step on the formatted output.
func AlignEquals(src []byte, depth int) []byte {
	lines := strings.Split(string(src), "\n")
	result := alignEqualsInLines(lines, depth)
	return []byte(strings.Join(result, "\n"))
}

// alignEqualsInLines processes lines and aligns = in consecutive single-line attribute groups.
func alignEqualsInLines(lines []string, _ int) []string {
	result := make([]string, len(lines))
	copy(result, lines)

	i := 0
	for i < len(lines) {
		// Try to start a group of consecutive single-line attributes at this line
		groupStart, groupEnd := findAttrGroup(lines, i)
		if groupEnd > groupStart {
			alignGroup(result, lines, groupStart, groupEnd)
			i = groupEnd + 1
		} else {
			i++
		}
	}
	return result
}

// findAttrGroup finds a group of consecutive single-line attribute lines
// starting at startIdx. Returns [start, end] inclusive, or start==end if no group.
func findAttrGroup(lines []string, startIdx int) (int, int) {
	if startIdx >= len(lines) {
		return startIdx, startIdx - 1
	}

	line := lines[startIdx]
	if !isSingleLineAttr(line) {
		return startIdx, startIdx - 1
	}

	groupStart := startIdx
	groupEnd := startIdx

	for j := startIdx + 1; j < len(lines); j++ {
		if isSingleLineAttr(lines[j]) && indentLevel(lines[j]) == indentLevel(lines[groupStart]) {
			groupEnd = j
		} else {
			break
		}
	}

	if groupEnd == groupStart {
		// Single attribute - no alignment needed
		return groupStart, groupStart - 1
	}
	return groupStart, groupEnd
}

// isSingleLineAttr returns true if the line is a single-line attribute assignment.
func isSingleLineAttr(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	if trimmed == "" || strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
		return false
	}
	if strings.HasSuffix(strings.TrimRight(trimmed, " \t"), "{") {
		return false
	}
	if strings.TrimRight(trimmed, " \t") == "}" {
		return false
	}
	return hasAttributeEquals(trimmed)
}

// indentLevel returns the number of leading spaces in a line.
func indentLevel(line string) int {
	count := 0
	for _, ch := range line {
		if ch == ' ' {
			count++
		} else if ch == '\t' {
			count += 2
		} else {
			break
		}
	}
	return count
}

// alignGroup aligns the = signs in lines[start..end] (inclusive).
func alignGroup(result []string, lines []string, start, end int) {
	// Find the key and value for each line
	type attrParts struct {
		indent string
		key    string
		value  string
	}

	parts := make([]attrParts, end-start+1)
	maxKeyLen := 0

	for i := start; i <= end; i++ {
		line := lines[i]
		// Find the = sign
		eqIdx := findEqualSign(line)
		if eqIdx < 0 {
			// Shouldn't happen, but skip
			continue
		}
		prefix := line[:eqIdx]
		value := strings.TrimLeft(line[eqIdx+1:], " \t")

		// Split prefix into indent + key
		trimmedPrefix := strings.TrimLeft(prefix, " \t")
		indent := line[:len(prefix)-len(trimmedPrefix)]
		key := strings.TrimRight(trimmedPrefix, " \t")

		if len(key) > maxKeyLen {
			maxKeyLen = len(key)
		}
		parts[i-start] = attrParts{indent: indent, key: key, value: value}
	}

	// Rewrite lines with aligned =
	for i := start; i <= end; i++ {
		p := parts[i-start]
		if p.key == "" {
			continue
		}
		padding := strings.Repeat(" ", maxKeyLen-len(p.key))
		result[i] = p.indent + p.key + padding + " = " + p.value
	}
}

// findEqualSign finds the index of the = sign in a line that denotes an attribute.
func findEqualSign(line string) int {
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if ch == '=' {
			// Ensure it's not ==, !=, <=, >=
			if i+1 < len(line) && line[i+1] == '=' {
				i++ // skip ==
				continue
			}
			if i > 0 {
				prev := line[i-1]
				if prev == '!' || prev == '<' || prev == '>' || prev == '=' {
					continue
				}
			}
			return i
		}
		// Skip strings
		if ch == '"' {
			i++
			for i < len(line) && line[i] != '"' {
				if line[i] == '\\' {
					i++
				}
				i++
			}
		}
	}
	return -1
}
