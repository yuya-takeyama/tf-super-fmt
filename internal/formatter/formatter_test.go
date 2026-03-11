package formatter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yuya-takeyama/tf-super-fmt/internal/formatter"
)

// goldenTest describes a testdata/golden pair.
type goldenTest struct {
	name  string
	input string
	want  string
}

// loadGoldenTests reads all *_input.tf / *_expected.tf pairs from testdata/golden.
func loadGoldenTests(t *testing.T) []goldenTest {
	t.Helper()
	dir := "../../testdata/golden"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cannot read testdata/golden: %v", err)
	}

	var tests []goldenTest
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, "_input.tf") {
			continue
		}
		base := strings.TrimSuffix(name, "_input.tf")
		inputPath := filepath.Join(dir, name)
		expectedPath := filepath.Join(dir, base+"_expected.tf")

		inputBytes, err := os.ReadFile(inputPath)
		if err != nil {
			t.Fatalf("cannot read %s: %v", inputPath, err)
		}
		expectedBytes, err := os.ReadFile(expectedPath)
		if err != nil {
			t.Fatalf("cannot read %s: %v", expectedPath, err)
		}
		tests = append(tests, goldenTest{
			name:  base,
			input: string(inputBytes),
			want:  string(expectedBytes),
		})
	}
	return tests
}

func TestFormat_Golden(t *testing.T) {
	tests := loadGoldenTests(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := formatter.Format(tt.name+".tf", []byte(tt.input))
			if err != nil {
				t.Fatalf("Format error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("Format(%q) mismatch\n--- want ---\n%s\n--- got ---\n%s",
					tt.name, tt.want, string(got))
			}
		})
	}
}

// TestFormat_Idempotent verifies that formatting already-formatted output
// produces the same result (i.e. format is stable).
func TestFormat_Idempotent(t *testing.T) {
	tests := loadGoldenTests(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First pass (may reformat)
			pass1, err := formatter.Format(tt.name+".tf", []byte(tt.input))
			if err != nil {
				t.Fatalf("Format pass1 error: %v", err)
			}
			// Second pass on the already-formatted output
			pass2, err := formatter.Format(tt.name+".tf", pass1)
			if err != nil {
				t.Fatalf("Format pass2 error: %v", err)
			}
			if string(pass1) != string(pass2) {
				t.Errorf("Format(%q) is not idempotent\n--- pass1 ---\n%s\n--- pass2 ---\n%s",
					tt.name, string(pass1), string(pass2))
			}
		})
	}
}

// TestFormat_MultiLineAttrIndent verifies that closing braces/brackets
// in multi-line attribute values keep their relative indentation.
func TestFormat_MultiLineAttrIndent(t *testing.T) {
	input := `resource "aws_instance" "web" {
  tags = merge(var.common_tags, {
    Name = "web"
  })
}
`
	got, err := formatter.Format("test.tf", []byte(input))
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	if string(got) != input {
		t.Errorf("multi-line attr closing brace indented wrongly\n--- want ---\n%s\n--- got ---\n%s", input, string(got))
	}
}

// TestFormat_CommentBlankLine verifies that blank lines before a block
// that has a leading comment are preserved.
func TestFormat_CommentBlankLine(t *testing.T) {
	input := `resource "aws_security_group" "web" {
  ingress {
    from_port = 443
  }

  # SSH is temporarily allowed
  ingress {
    from_port = 22
  }
}
`
	got, err := formatter.Format("test.tf", []byte(input))
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	if string(got) != input {
		t.Errorf("blank line before comment-led block was dropped\n--- want ---\n%s\n--- got ---\n%s", input, string(got))
	}
}

// TestFormat_ListClosingBracket verifies that closing ] in ignore_changes
// keeps its relative indentation.
func TestFormat_ListClosingBracket(t *testing.T) {
	input := `resource "foo" "bar" {
  lifecycle {
    ignore_changes = [
      tags,
      user_data,
    ]
  }
}
`
	got, err := formatter.Format("test.tf", []byte(input))
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	if string(got) != input {
		t.Errorf("closing ] in list attribute indented wrongly\n--- want ---\n%s\n--- got ---\n%s", input, string(got))
	}
}

// TestFormat_NestedBlockAttr verifies that closing } in a block-like
// attribute value (e.g. required_providers) keeps its indentation.
func TestFormat_NestedBlockAttr(t *testing.T) {
	input := `terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`
	got, err := formatter.Format("test.tf", []byte(input))
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	if string(got) != input {
		t.Errorf("nested block-like attr closing brace indented wrongly\n--- want ---\n%s\n--- got ---\n%s", input, string(got))
	}
}

// TestFormat_CommentBeforeFirstBlock verifies that a leading comment on the
// very first line of the file is preserved after formatting.
// Regression test for: resource の直前のコメントが消える bug.
func TestFormat_CommentBeforeFirstBlock(t *testing.T) {
	input := "# ALB\nresource \"aws_security_group\" \"alb\" {\n  name = \"alb\"\n}\n"
	want := "# ALB\nresource \"aws_security_group\" \"alb\" {\n  name = \"alb\"\n}\n"

	got, err := formatter.Format("test.tf", []byte(input))
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	if string(got) != want {
		t.Errorf("leading comment was dropped\n--- want ---\n%s\n--- got ---\n%s", want, string(got))
	}
}

// TestFormat_AlignSkipsListAttr verifies that an attribute whose value opens a
// list (e.g. "Statement = [") does not distort the = alignment of preceding
// single-line attributes.
// Regression test for: list が次の行にあるときの = の位置 bug.
func TestFormat_AlignSkipsListAttr(t *testing.T) {
	// Both Version and Statement appear as raw text inside the multi-line
	// `policy` attribute value. AlignEquals must not pad "Version" to align
	// with "Statement" because "Statement = [" opens a multi-line list.
	input := "resource \"aws_s3_bucket_policy\" \"alb_logs\" {\n  bucket = aws_s3_bucket.alb_logs.id\n  policy = jsonencode({\n    Version = \"2012-10-17\"\n    Statement = [\n      {\n        Effect = \"Allow\"\n      }\n    ]\n  })\n}\n"
	want := input // no change expected

	got, err := formatter.Format("test.tf", []byte(input))
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	if string(got) != want {
		t.Errorf("= alignment incorrectly padded Version\n--- want ---\n%s\n--- got ---\n%s", want, string(got))
	}
}
