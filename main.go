// Command idea is a tiny CLI for capturing numbered ideas in a markdown file.
package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const header = "# Ideas"

const usage = `Usage:
  idea "some text"      Append an idea
  idea -l               List ideas
  idea -r <spec>        Remove ideas by number (e.g. 1, 1-3, 1,5,6, 1-3,5,7-9)
  idea -h               Show this help

Ideas are stored in ~/.ideas.md (override with $IDEAS_FILE).
`

var itemRe = regexp.MustCompile(`^(\d+)\.\s?(.*)$`)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "idea:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Print(usage)
		return nil
	}

	path, err := ideasPath()
	if err != nil {
		return err
	}

	switch args[0] {
	case "-h", "--help", "help":
		fmt.Print(usage)
		return nil
	case "-l", "--list":
		if len(args) > 1 {
			return errors.New("-l takes no arguments")
		}
		return list(path)
	case "-r", "--remove":
		if len(args) != 2 {
			return errors.New("-r requires exactly one spec, e.g. -r 1-3,5")
		}
		return remove(path, args[1])
	case "--":
		args = args[1:]
	default:
		if strings.HasPrefix(args[0], "-") {
			return fmt.Errorf("unknown flag %q (use -- before text starting with '-')", args[0])
		}
	}

	return add(path, strings.Join(args, " "))
}

// ideasPath returns $IDEAS_FILE if set, otherwise ~/.ideas.md.
func ideasPath() (string, error) {
	if p := os.Getenv("IDEAS_FILE"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locating home directory: %w", err)
	}
	return filepath.Join(home, ".ideas.md"), nil
}

// load reads the ideas file and returns the idea texts in order.
// A missing file yields no ideas and no error.
func load(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var ideas []string
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		if m := itemRe.FindStringSubmatch(strings.TrimRight(sc.Text(), "\r")); m != nil {
			ideas = append(ideas, m[2])
		}
	}
	return ideas, sc.Err()
}

// save writes the header and the ideas, numbered from 1, atomically.
func save(path string, ideas []string) error {
	var b strings.Builder
	b.WriteString(header + "\n")
	for i, idea := range ideas {
		fmt.Fprintf(&b, "%d. %s\n", i+1, idea)
	}

	// Write through symlinks so a linked ideas file stays linked.
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}

	mode := fs.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".ideas-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename

	if _, err := tmp.WriteString(b.String()); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func add(path, text string) error {
	// Keep each idea on a single line so numbering stays intact.
	text = strings.Join(strings.Fields(text), " ")
	if text == "" {
		return errors.New("idea text is empty")
	}

	ideas, err := load(path)
	if err != nil {
		return err
	}
	ideas = append(ideas, text)
	if err := save(path, ideas); err != nil {
		return err
	}
	fmt.Printf("Added idea #%d\n", len(ideas))
	return nil
}

func list(path string) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		fmt.Println("No ideas yet.")
		return nil
	}
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(data)
	return err
}

func remove(path, spec string) error {
	ideas, err := load(path)
	if err != nil {
		return err
	}
	if len(ideas) == 0 {
		return errors.New("no ideas to remove")
	}

	targets, err := parseSpec(spec, len(ideas))
	if err != nil {
		return err
	}

	kept := make([]string, 0, len(ideas)-len(targets))
	for i, idea := range ideas {
		if !targets[i+1] {
			kept = append(kept, idea)
		}
	}
	if err := save(path, kept); err != nil {
		return err
	}

	removed := make([]int, 0, len(targets))
	for n := range targets {
		removed = append(removed, n)
	}
	sort.Ints(removed)
	strs := make([]string, len(removed))
	for i, n := range removed {
		strs[i] = strconv.Itoa(n)
	}
	fmt.Printf("Removed %d idea(s): %s\n", len(removed), strings.Join(strs, ", "))
	return nil
}

// parseSpec parses a removal spec like "1-3,5,7-9" into a set of idea
// numbers, validating each against the range 1..max.
func parseSpec(spec string, max int) (map[int]bool, error) {
	set := make(map[int]bool)
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("invalid spec %q: empty element", spec)
		}

		lo, hi := part, part
		if a, b, ok := strings.Cut(part, "-"); ok {
			lo, hi = strings.TrimSpace(a), strings.TrimSpace(b)
		}
		start, err := parseNum(lo, max)
		if err != nil {
			return nil, err
		}
		end, err := parseNum(hi, max)
		if err != nil {
			return nil, err
		}
		if start > end {
			return nil, fmt.Errorf("invalid range %q: start is greater than end", part)
		}
		for n := start; n <= end; n++ {
			set[n] = true
		}
	}
	return set, nil
}

func parseNum(s string, max int) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid number %q", s)
	}
	if n < 1 || n > max {
		return 0, fmt.Errorf("idea %d does not exist (valid range: 1-%d)", n, max)
	}
	return n, nil
}
