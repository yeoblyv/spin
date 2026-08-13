// Package hosts manages the delimited, spin-owned block of local-domain
// entries inside an OS hosts file, without disturbing anything a user or
// another tool wrote outside that block.
package hosts

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	beginMarker = "# spin:begin - managed by spin, do not edit by hand"
	endMarker   = "# spin:end"
)

// Render produces spin's delimited block for the given domain -> IP
// mappings, sorted by domain for a stable, diffable rendering.
func Render(entries map[string]string) string {
	domains := make([]string, 0, len(entries))
	for d := range entries {
		domains = append(domains, d)
	}
	sort.Strings(domains)

	var b strings.Builder
	b.WriteString(beginMarker)
	b.WriteString("\n")
	for _, d := range domains {
		fmt.Fprintf(&b, "%s %s\n", entries[d], d)
	}
	b.WriteString(endMarker)
	b.WriteString("\n")
	return b.String()
}

// Upsert replaces spin's block inside content with a fresh rendering of
// entries, appending the block if content has none yet. Everything outside
// the block - a user's own entries, another tool's block - is untouched.
func Upsert(content string, entries map[string]string) string {
	before, after, found := splitBlock(content)
	block := Render(entries)

	if !found {
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		return content + block
	}

	return before + block + after
}

// Remove strips spin's block from content entirely, leaving everything
// else untouched. content without a block is returned unchanged.
func Remove(content string) string {
	before, after, found := splitBlock(content)
	if !found {
		return content
	}
	return before + after
}

// splitBlock locates spin's begin/end-delimited block and returns the
// content before it and after it. found is false when no complete block is
// present, in which case before equals content and after is empty.
func splitBlock(content string) (before, after string, found bool) {
	startIdx := strings.Index(content, beginMarker)
	if startIdx == -1 {
		return content, "", false
	}

	relativeEnd := strings.Index(content[startIdx:], endMarker)
	if relativeEnd == -1 {
		return content, "", false
	}
	endIdx := startIdx + relativeEnd + len(endMarker)

	// Consume a single trailing newline after the end marker, if present,
	// so re-inserting the block doesn't accumulate a blank line each time.
	if endIdx < len(content) && content[endIdx] == '\n' {
		endIdx++
	}

	return content[:startIdx], content[endIdx:], true
}

// Apply reads the hosts file at path, upserts spin's block for entries, and
// writes the result back, preserving the file's existing permissions (or
// 0o644 for a file that does not exist yet).
func Apply(path string, entries map[string]string) error {
	content, mode, err := readWithMode(path)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(Upsert(content, entries)), mode); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// Clear reads the hosts file at path and removes spin's block, if present.
func Clear(path string) error {
	content, mode, err := readWithMode(path)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(Remove(content)), mode); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

func readWithMode(path string) (string, os.FileMode, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", 0o644, nil
	}
	if err != nil {
		return "", 0, fmt.Errorf("reading %s: %w", path, err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", 0, fmt.Errorf("checking %s: %w", path, err)
	}
	return string(data), info.Mode().Perm(), nil
}
