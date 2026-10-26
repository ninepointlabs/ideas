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
  idea --version        Show the version

Ideas are stored in ~/Documents/Ideas/ideas.md (override with $IDEAS_FILE).
`

var itemRe = regexp.MustCompile(`^(\d+)\.\s?(.*)$`)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

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
	if args[0] == "--version" {
		fmt.Println("idea", version)
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

// ideasPath returns $IDEAS_FILE if set, otherwise ~/Documents/Ideas/ideas.md.
// If the old location (~/.ideas.md) exists, it is moved to the new default.
// If the new file already exists (reinstall, Syncthing), it is used as-is.
func ideasPath() (string, error) {
	if p := os.Getenv("IDEAS_FILE"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locating home directory: %w", err)
	}
	newDir := filepath.Join(home, "Documents", "Ideas")
	newPath := filepath.Join(newDir, "ideas.md")
	oldPath := filepath.Join(home, ".ideas.md")

	// New file already exists — reinstall or Syncthing, use it directly.
	if _, err := os.Stat(newPath); err == nil {
		return newPath, nil
	}

	// Migrate from old location if it exists.
	if _, err := os.Stat(oldPath); err == nil {
		if err := os.MkdirAll(newDir, 0o755); err != nil {
			return "", fmt.Errorf("creating Ideas folder: %w", err)
		}
		if err := os.Rename(oldPath, newPath); err != nil {
			return "", fmt.Errorf("migrating ideas from %s: %w", oldPath, err)
		}
		fmt.Fprintf(os.Stderr, "idea: moved ideas to %s\n", newPath)
		return newPath, nil
	}

	// Nothing exists yet — create the directory for the first run.
	if err := os.MkdirAll(newDir, 0o755); err != nil {
		return "", fmt.Errorf("creating Ideas folder: %w", err)
	}
	return newPath, nil
}

// line is one line of the ideas file. Numbered lines are ideas; every other
// line (the header, headings, notes) is kept exactly as it was read.
type line struct {
	text   string // raw text of a line that isn't an idea
	idea   string // idea text, without its number
	isIdea bool
}

// load reads the ideas file into lines. A missing or empty file yields just
// the header.
func load(path string) ([]line, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) || (err == nil && len(data) == 0) {
		return []line{{text: header}}, nil
	}
	if err != nil {
		return nil, err
	}

	var lines []line
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		if m := itemRe.FindStringSubmatch(strings.TrimRight(sc.Text(), "\r")); m != nil {
			lines = append(lines, line{idea: m[2], isIdea: true})
		} else {
			lines = append(lines, line{text: sc.Text()})
		}
	}
	return lines, sc.Err()
}

// countIdeas returns the number of idea lines.
func countIdeas(lines []line) int {
	n := 0
	for _, l := range lines {
		if l.isIdea {
			n++
		}
	}
	return n
}

// save writes the lines atomically, numbering the ideas from 1.
func save(path string, lines []line) error {
	var b strings.Builder
	n := 0
	for _, l := range lines {
		if l.isIdea {
			n++
			fmt.Fprintf(&b, "%d. %s\n", n, l.idea)
		} else {
			b.WriteString(l.text + "\n")
		}
	}

	// Write through symlinks so a linked ideas file stays linked.
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}

	mode := fs.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
		// Respect a read-only file instead of replacing it via rename.
		f, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				return fmt.Errorf("%s is read-only; not changing it", path)
			}
			return err
		}
		f.Close()
	}

	// Create the parent folder, e.g. for a new $IDEAS_FILE location.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating folder for %s: %w", path, err)
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

	lines, err := load(path)
	if err != nil {
		return err
	}

	// Insert after the last idea so trailing notes stay at the end.
	at := len(lines)
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i].isIdea {
			at = i + 1
			break
		}
	}
	lines = append(lines[:at], append([]line{{idea: text, isIdea: true}}, lines[at:]...)...)
	if err := save(path, lines); err != nil {
		return err
	}
	fmt.Printf("Added idea #%d\n", countIdeas(lines))
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
	lines, err := load(path)
	if err != nil {
		return err
	}
	count := countIdeas(lines)
	if count == 0 {
		return errors.New("no ideas to remove")
	}

	targets, err := parseSpec(spec, count)
	if err != nil {
		return err
	}

	kept := make([]line, 0, len(lines))
	n := 0
	for _, l := range lines {
		if l.isIdea {
			n++
			if targets[n] {
				continue
			}
		}
		kept = append(kept, l)
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
		start, err := parseNum(lo)
		if err != nil {
			return nil, err
		}
		end, err := parseNum(hi)
		if err != nil {
			return nil, err
		}
		if start > end {
			return nil, fmt.Errorf("invalid range %q: the range is reversed (did you mean %d-%d?)", part, end, start)
		}
		for _, n := range []int{start, end} {
			if n < 1 || n > max {
				return nil, fmt.Errorf("idea %d does not exist (valid range: 1-%d)", n, max)
			}
		}
		for n := start; n <= end; n++ {
			set[n] = true
		}
	}
	return set, nil
}

func parseNum(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid number %q", s)
	}
	return n, nil
}
