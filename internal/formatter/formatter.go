package formatter

// Format parses and reformats the given HCL source bytes.
// path is used for error messages.
func Format(path string, src []byte) ([]byte, error) {
	fm, err := Parse(path, src)
	if err != nil {
		return nil, err
	}

	formatted := Rewrite(fm)

	// Apply = alignment as a post-processing step
	formatted = AlignEquals(formatted, 0)

	// Apply original EOL
	formatted = ApplyEOL(formatted, fm.EOL)

	return formatted, nil
}
