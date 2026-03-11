package formatter

import (
	"strings"
	"testing"
)

// TestIsSingleLineAttr_MultiLineValueExcluded verifies that attributes whose value
// opens a multi-line expression are excluded from = alignment groups.
// Regression test for: list が次の行にあるときの = の位置 bug.
func TestIsSingleLineAttr_MultiLineValueExcluded(t *testing.T) {
	cases := []struct {
		line string
		want bool
		desc string
	}{
		{"  Statement = [", false, "list open"},
		{"  cidr_blocks = [", false, "list open with underscores"},
		{"  tags = merge(", false, "function call open"},
		{"  policy = jsonencode(", false, "jsonencode call open"},
		{"  name = \"example\"", true, "plain string value"},
		{"  count = 1", true, "integer value"},
		{"  enabled = true", true, "boolean value"},
		{"  cidr_blocks = [\"0.0.0.0/0\"]", true, "single-line list (closes on same line)"},
		{"  tags = merge(a, b)", true, "single-line function call (closes on same line)"},
	}
	for _, tc := range cases {
		got := isSingleLineAttr(tc.line)
		if got != tc.want {
			t.Errorf("isSingleLineAttr(%q) [%s] = %v, want %v", tc.line, tc.desc, got, tc.want)
		}
	}
}

// TestAlignEquals_SkipsListOpenAttr verifies that a line like "Statement = ["
// is not pulled into the alignment group, so preceding single-line attributes
// are not incorrectly padded.
func TestAlignEquals_SkipsListOpenAttr(t *testing.T) {
	input := strings.Join([]string{
		"    Version = \"2012-10-17\"",
		"    Statement = [",
		"      \"allow\",",
		"    ]",
		"",
	}, "\n")

	// Statement = [ is excluded → Version is the only element → no alignment.
	expected := strings.Join([]string{
		"    Version = \"2012-10-17\"",
		"    Statement = [",
		"      \"allow\",",
		"    ]",
		"",
	}, "\n")

	got := string(AlignEquals([]byte(input), 0))
	if got != expected {
		t.Errorf("AlignEquals() mismatch:\ngot:\n%s\nwant:\n%s", got, expected)
	}
}

// TestAlignEquals_SplitsGroupAtListAttr verifies that attributes before and
// after a list-opening attribute form independent alignment groups.
func TestAlignEquals_SplitsGroupAtListAttr(t *testing.T) {
	input := strings.Join([]string{
		"  type = \"ingress\"",
		"  protocol = \"tcp\"",
		"  cidr_blocks = [",
		"    \"0.0.0.0/0\",",
		"  ]",
		"  from_port = 80",
		"  to_port = 80",
		"",
	}, "\n")

	// "type" and "protocol" align together (max key = "protocol" = 8 chars).
	// "cidr_blocks = [" is excluded.
	// "from_port" and "to_port" align together (max key = "from_port" = 9 chars).
	expected := strings.Join([]string{
		"  type     = \"ingress\"",
		"  protocol = \"tcp\"",
		"  cidr_blocks = [",
		"    \"0.0.0.0/0\",",
		"  ]",
		"  from_port = 80",
		"  to_port   = 80",
		"",
	}, "\n")

	got := string(AlignEquals([]byte(input), 0))
	if got != expected {
		t.Errorf("AlignEquals() group split mismatch:\ngot:\n%s\nwant:\n%s", got, expected)
	}
}
