package discover

import (
	"os"
	"path/filepath"
	"strings"
)

// targetExtensions lists the file patterns to include.
var targetExtensions = []string{
	".tf",
	".tfvars",
	".tftest.hcl",
	".tfmock.hcl",
}

// targetFilenames lists specific filenames to include.
var targetFilenames = []string{
	".terraform.lock.hcl",
}

// excludeDirs lists directory names to skip during recursive traversal.
var excludeDirs = []string{
	".terraform",
	".git",
}

// isTargetFile returns true if the given filename should be formatted.
func isTargetFile(name string) bool {
	for _, fname := range targetFilenames {
		if name == fname {
			return true
		}
	}
	for _, ext := range targetExtensions {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

// isExcludedDir returns true if the directory should be skipped.
func isExcludedDir(name string) bool {
	for _, d := range excludeDirs {
		if name == d {
			return true
		}
	}
	return false
}

// Find returns a list of HCL/Terraform files in the given targets.
// targets can be file paths or directories.
// If recursive is true, subdirectories are walked (excluding .terraform/ and .git/).
func Find(targets []string, recursive bool) ([]string, error) {
	var result []string
	seen := make(map[string]bool)

	for _, target := range targets {
		info, err := os.Stat(target)
		if err != nil {
			return nil, err
		}

		if info.IsDir() {
			files, err := findInDir(target, recursive)
			if err != nil {
				return nil, err
			}
			for _, f := range files {
				abs, err := filepath.Abs(f)
				if err != nil {
					return nil, err
				}
				if !seen[abs] {
					seen[abs] = true
					result = append(result, f)
				}
			}
		} else {
			if isTargetFile(info.Name()) {
				abs, err := filepath.Abs(target)
				if err != nil {
					return nil, err
				}
				if !seen[abs] {
					seen[abs] = true
					result = append(result, target)
				}
			}
		}
	}

	return result, nil
}

// findInDir finds target files in a directory.
func findInDir(dir string, recursive bool) ([]string, error) {
	var result []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(dir, name)

		if entry.IsDir() {
			if recursive && !isExcludedDir(name) {
				subFiles, err := findInDir(path, recursive)
				if err != nil {
					return nil, err
				}
				result = append(result, subFiles...)
			}
		} else {
			if isTargetFile(name) {
				result = append(result, path)
			}
		}
	}

	return result, nil
}
