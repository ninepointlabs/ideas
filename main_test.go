package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveKeepsOtherLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ideas.md")
	in := "# Ideas\n\n## Work\n1. one\n2. two\nnote about two\n3. three\n\nTrailing note\n"
	if err := os.WriteFile(path, []byte(in), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := remove(path, "2"); err != nil {
		t.Fatal(err)
	}
	if err := add(path, "four"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	want := "# Ideas\n\n## Work\n1. one\nnote about two\n2. three\n3. four\n\nTrailing note\n"
	if string(got) != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestAddCreatesFileAndFolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new", "dir", "ideas.md")
	if err := add(path, "first"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if want := "# Ideas\n1. first\n"; string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestReadOnlyFileIsLeftAlone(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write read-only files")
	}
	path := filepath.Join(t.TempDir(), "ideas.md")
	in := "# Ideas\n1. one\n"
	if err := os.WriteFile(path, []byte(in), 0o444); err != nil {
		t.Fatal(err)
	}
	err := add(path, "two")
	if err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("want read-only error, got %v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != in {
		t.Errorf("file changed: %q", got)
	}
}

func TestParseSpecReversedRange(t *testing.T) {
	// Reported as reversed even when the start is out of range.
	_, err := parseSpec("3-1", 2)
	if err == nil || !strings.Contains(err.Error(), "reversed") {
		t.Fatalf("want reversed-range error, got %v", err)
	}
}
