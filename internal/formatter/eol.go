package formatter

import "bytes"

// DetectEOL detects the line ending style used in the source.
// If all line endings are CRLF → EOLCRLF
// If all line endings are LF → EOLLF
// Mixed → EOLLF (normalize to LF)
func DetectEOL(src []byte) EOLKind {
	if len(src) == 0 {
		return EOLLF
	}

	crlfCount := bytes.Count(src, []byte("\r\n"))
	lfCount := bytes.Count(src, []byte("\n"))

	if crlfCount == 0 {
		return EOLLF
	}
	if crlfCount == lfCount {
		// Every \n is preceded by \r, so all line endings are CRLF
		return EOLCRLF
	}
	// Mixed
	return EOLLF
}

// NormalizeToLF replaces all CRLF with LF.
func NormalizeToLF(src []byte) []byte {
	return bytes.ReplaceAll(src, []byte("\r\n"), []byte("\n"))
}

// ApplyEOL converts LF line endings to the target EOL style.
func ApplyEOL(src []byte, eol EOLKind) []byte {
	if eol == EOLCRLF {
		// Replace all LF that are not preceded by CR
		return bytes.ReplaceAll(src, []byte("\n"), []byte("\r\n"))
	}
	return src
}
