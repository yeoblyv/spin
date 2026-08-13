package hosts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderIsSortedByDomain(t *testing.T) {
	got := Render(map[string]string{
		"zebra.test": "127.0.0.1",
		"apple.test": "127.0.0.1",
	})

	appleIdx := strings.Index(got, "apple.test")
	zebraIdx := strings.Index(got, "zebra.test")
	if appleIdx == -1 || zebraIdx == -1 {
		t.Fatalf("Render() = %q, want both domains present", got)
	}
	if appleIdx > zebraIdx {
		t.Errorf("Render() = %q, want apple.test before zebra.test", got)
	}
	if !strings.HasPrefix(got, beginMarker) {
		t.Errorf("Render() = %q, want it to start with the begin marker", got)
	}
	if !strings.HasSuffix(got, endMarker+"\n") {
		t.Errorf("Render() = %q, want it to end with the end marker", got)
	}
}

func TestUpsertAppendsBlockWhenAbsent(t *testing.T) {
	original := "127.0.0.1 localhost\n"
	got := Upsert(original, map[string]string{"blog.test": "127.0.0.1"})

	if !strings.HasPrefix(got, original) {
		t.Errorf("Upsert() = %q, want the original content preserved at the start", got)
	}
	if !strings.Contains(got, "blog.test") {
		t.Errorf("Upsert() = %q, want it to contain the new domain", got)
	}
}

func TestUpsertReplacesExistingBlockOnly(t *testing.T) {
	original := "127.0.0.1 localhost\n" +
		beginMarker + "\n127.0.0.1 old.test\n" + endMarker + "\n" +
		"10.0.0.1 other-tool-entry\n"

	got := Upsert(original, map[string]string{"new.test": "127.0.0.1"})

	if strings.Contains(got, "old.test") {
		t.Errorf("Upsert() = %q, want the stale domain removed", got)
	}
	if !strings.Contains(got, "new.test") {
		t.Errorf("Upsert() = %q, want the new domain present", got)
	}
	if !strings.Contains(got, "127.0.0.1 localhost") {
		t.Errorf("Upsert() = %q, want content before the block preserved", got)
	}
	if !strings.Contains(got, "10.0.0.1 other-tool-entry") {
		t.Errorf("Upsert() = %q, want content after the block preserved", got)
	}
}

func TestUpsertIsIdempotent(t *testing.T) {
	entries := map[string]string{"blog.test": "127.0.0.1"}
	once := Upsert("", entries)
	twice := Upsert(once, entries)

	if once != twice {
		t.Errorf("Upsert() is not idempotent:\nonce  = %q\ntwice = %q", once, twice)
	}
}

func TestRemoveStripsBlockOnly(t *testing.T) {
	original := "127.0.0.1 localhost\n" +
		beginMarker + "\n127.0.0.1 blog.test\n" + endMarker + "\n" +
		"10.0.0.1 other-tool-entry\n"

	got := Remove(original)

	if strings.Contains(got, "blog.test") {
		t.Errorf("Remove() = %q, want the block gone", got)
	}
	if !strings.Contains(got, "127.0.0.1 localhost") || !strings.Contains(got, "10.0.0.1 other-tool-entry") {
		t.Errorf("Remove() = %q, want surrounding content preserved", got)
	}
}

func TestRemoveWithoutBlockIsNoop(t *testing.T) {
	original := "127.0.0.1 localhost\n"
	if got := Remove(original); got != original {
		t.Errorf("Remove() = %q, want it unchanged when no block is present", got)
	}
}

func TestApplyAndClearRoundTripOnDisk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hosts")
	if err := os.WriteFile(path, []byte("127.0.0.1 localhost\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := Apply(path, map[string]string{"blog.test": "127.0.0.1"}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	content := readFile(t, path)
	if !strings.Contains(content, "blog.test") {
		t.Errorf("hosts file = %q, want it to contain the applied domain", content)
	}
	if !strings.Contains(content, "127.0.0.1 localhost") {
		t.Errorf("hosts file = %q, want the pre-existing line preserved", content)
	}

	if err := Clear(path); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}
	content = readFile(t, path)
	if strings.Contains(content, "blog.test") {
		t.Errorf("hosts file = %q, want the block removed after Clear()", content)
	}
	if !strings.Contains(content, "127.0.0.1 localhost") {
		t.Errorf("hosts file = %q, want the pre-existing line still preserved", content)
	}
}

func TestApplyOnMissingFileCreatesIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hosts")

	if err := Apply(path, map[string]string{"blog.test": "127.0.0.1"}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	content := readFile(t, path)
	if !strings.Contains(content, "blog.test") {
		t.Errorf("hosts file = %q, want it to contain the applied domain", content)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(data)
}
