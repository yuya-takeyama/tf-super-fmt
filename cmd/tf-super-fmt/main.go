package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/yuya-takeyama/tf-super-fmt/internal/diff"
	"github.com/yuya-takeyama/tf-super-fmt/internal/discover"
	"github.com/yuya-takeyama/tf-super-fmt/internal/formatter"
)

// Exit codes
const (
	ExitOK       = 0
	ExitCheck    = 1 // check found changes
	ExitParse    = 2 // parse error
	ExitInternal = 3 // internal error
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		recursive = flag.Bool("recursive", false, "recurse into subdirectories")
		check     = flag.Bool("check", false, "exit 1 if any file needs changes")
		showDiff  = flag.Bool("diff", false, "show unified diff")
		write     = flag.Bool("write", true, "write formatted files")
		list      = flag.Bool("list", true, "list files that differ")
		noColor   = flag.Bool("no-color", false, "disable color output")
	)
	flag.Parse()

	_ = noColor // color support is optional

	args := flag.Args()

	// Handle stdin mode
	if len(args) == 1 && args[0] == "-" {
		return processStdin()
	}

	// Determine target directories/files
	targets := args
	if len(targets) == 0 {
		targets = []string{"."}
	}

	// Discover files
	files, err := discover.Find(targets, *recursive)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error discovering files: %v\n", err)
		return ExitInternal
	}

	if len(files) == 0 {
		return ExitOK
	}

	hasChanges := false
	hasErrors := false

	for _, file := range files {
		changed, parseErr, internalErr := processFile(file, *write, *check, *showDiff, *list)
		if internalErr != nil {
			fmt.Fprintf(os.Stderr, "error processing %s: %v\n", file, internalErr)
			hasErrors = true
		} else if parseErr != nil {
			fmt.Fprintf(os.Stderr, "parse error in %s: %v\n", file, parseErr)
			hasErrors = true
		} else if changed {
			hasChanges = true
		}
	}

	if hasErrors {
		return ExitParse
	}
	if *check && hasChanges {
		return ExitCheck
	}
	return ExitOK
}

// processStdin reads from stdin, formats, and writes to stdout.
func processStdin() int {
	src, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
		return ExitInternal
	}

	formatted, err := formatter.Format("<stdin>", src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		return ExitParse
	}

	_, err = os.Stdout.Write(formatted)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing stdout: %v\n", err)
		return ExitInternal
	}

	return ExitOK
}

// processFile processes a single file.
// Returns (changed, parseErr, internalErr).
func processFile(path string, write, check, showDiff, list bool) (bool, error, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return false, nil, fmt.Errorf("reading file: %w", err)
	}

	formatted, err := formatter.Format(path, src)
	if err != nil {
		return false, err, nil
	}

	changed := !bytes.Equal(src, formatted)

	if !changed {
		return false, nil, nil
	}

	if list {
		relPath, err := filepath.Rel(".", path)
		if err != nil {
			relPath = path
		}
		fmt.Println(relPath)
	}

	if showDiff {
		d := diff.Unified(path, src, formatted)
		if d != "" {
			fmt.Print(d)
		}
	}

	if write && !check {
		if err := os.WriteFile(path, formatted, 0644); err != nil {
			return true, nil, fmt.Errorf("writing file: %w", err)
		}
	}

	return true, nil, nil
}
