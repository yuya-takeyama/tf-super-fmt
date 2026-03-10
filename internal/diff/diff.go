package diff

import (
	"fmt"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// Unified generates a unified diff between original and formatted content.
// filename is used in the diff header.
func Unified(filename string, original, formatted []byte) string {
	if string(original) == string(formatted) {
		return ""
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(original), string(formatted), false)
	dmp.DiffCleanupSemantic(diffs)

	return formatUnifiedDiff(filename, diffs)
}

// formatUnifiedDiff formats diffs as a unified diff output.
func formatUnifiedDiff(filename string, diffs []diffmatchpatch.Diff) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("--- %s\n", filename))
	sb.WriteString(fmt.Sprintf("+++ %s\n", filename))

	// Convert diffs to line-based hunks
	origLines := []string{}
	newLines := []string{}
	origLine := 1
	newLine := 1

	type hunkLine struct {
		kind rune // ' ', '+', '-'
		text string
		orig int
		new  int
	}

	var allLines []hunkLine

	for _, d := range diffs {
		lines := splitDiffText(d.Text)
		switch d.Type {
		case diffmatchpatch.DiffEqual:
			for _, l := range lines {
				allLines = append(allLines, hunkLine{' ', l, origLine, newLine})
				origLine++
				newLine++
			}
		case diffmatchpatch.DiffDelete:
			for _, l := range lines {
				allLines = append(allLines, hunkLine{'-', l, origLine, -1})
				origLine++
			}
		case diffmatchpatch.DiffInsert:
			for _, l := range lines {
				allLines = append(allLines, hunkLine{'+', l, -1, newLine})
				newLine++
			}
		}
	}

	_ = origLines
	_ = newLines

	// Group into hunks (context of 3 lines)
	const context = 3
	type hunk struct {
		lines     []hunkLine
		origStart int
		newStart  int
	}

	var hunks []hunk
	i := 0
	for i < len(allLines) {
		if allLines[i].kind == ' ' {
			i++
			continue
		}
		// Found a changed line, build a hunk
		hunkStart := i - context
		if hunkStart < 0 {
			hunkStart = 0
		}

		hunkEnd := i + 1
		for hunkEnd < len(allLines) {
			if allLines[hunkEnd].kind != ' ' {
				hunkEnd++
			} else {
				// Count trailing context
				trailEnd := hunkEnd + context
				if trailEnd > len(allLines) {
					trailEnd = len(allLines)
				}
				// Check if there's another change within context
				hasMore := false
				for k := hunkEnd; k < trailEnd; k++ {
					if allLines[k].kind != ' ' {
						hasMore = true
						break
					}
				}
				if hasMore {
					hunkEnd = trailEnd
				} else {
					hunkEnd = hunkEnd + context
					if hunkEnd > len(allLines) {
						hunkEnd = len(allLines)
					}
					break
				}
			}
		}
		if hunkEnd > len(allLines) {
			hunkEnd = len(allLines)
		}

		// Determine orig/new start lines
		origStart := 1
		newStart := 1
		for _, l := range allLines[hunkStart:] {
			if l.orig > 0 {
				origStart = l.orig
				break
			}
		}
		for _, l := range allLines[hunkStart:] {
			if l.new > 0 {
				newStart = l.new
				break
			}
		}

		hunks = append(hunks, hunk{
			lines:     allLines[hunkStart:hunkEnd],
			origStart: origStart,
			newStart:  newStart,
		})

		i = hunkEnd
	}

	for _, h := range hunks {
		// Count orig and new line counts
		origCount := 0
		newCount := 0
		for _, l := range h.lines {
			if l.kind == ' ' || l.kind == '-' {
				origCount++
			}
			if l.kind == ' ' || l.kind == '+' {
				newCount++
			}
		}

		sb.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
			h.origStart, origCount, h.newStart, newCount))

		for _, l := range h.lines {
			sb.WriteRune(l.kind)
			sb.WriteString(l.text)
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

// splitDiffText splits a diff text into lines, preserving line content without newlines.
func splitDiffText(text string) []string {
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	// Remove trailing empty element
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
