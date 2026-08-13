package registry

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// newSpiderProject creates a directory containing a bootstrap.php file, so
// IsSpiderProject recognizes it.
func newSpiderProject(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bootstrap.php"), []byte("<?php\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(bootstrap.php) error = %v", err)
	}
	return dir
}

func TestIsSpiderProjectDetectsBootstrapPHP(t *testing.T) {
	dir := newSpiderProject(t, "app")
	if !IsSpiderProject(dir) {
		t.Errorf("IsSpiderProject(%s) = false, want true", dir)
	}
}

func TestIsSpiderProjectDetectsComposerPackageName(t *testing.T) {
	dir := t.TempDir()
	composer := `{"name": "spider/framework"}`
	if err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(composer), 0o644); err != nil {
		t.Fatalf("WriteFile(composer.json) error = %v", err)
	}
	if !IsSpiderProject(dir) {
		t.Errorf("IsSpiderProject(%s) = false, want true", dir)
	}
}

func TestIsSpiderProjectRejectsUnrelatedDirectory(t *testing.T) {
	dir := t.TempDir()
	if IsSpiderProject(dir) {
		t.Errorf("IsSpiderProject(%s) = true, want false", dir)
	}
}

func TestIsSpiderProjectRejectsUnrelatedComposerPackage(t *testing.T) {
	dir := t.TempDir()
	composer := `{"name": "some/other-package"}`
	if err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(composer), 0o644); err != nil {
		t.Fatalf("WriteFile(composer.json) error = %v", err)
	}
	if IsSpiderProject(dir) {
		t.Errorf("IsSpiderProject(%s) = true, want false", dir)
	}
}

func TestAddRegistersAndDerivesName(t *testing.T) {
	dir := newSpiderProject(t, "blog")
	r, _ := Load(filepath.Join(t.TempDir(), "registry.json"))

	entry, err := r.Add("", dir)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if entry.Name != "blog" {
		t.Errorf("entry.Name = %q, want %q", entry.Name, "blog")
	}
	if entry.Path != dir {
		t.Errorf("entry.Path = %q, want %q", entry.Path, dir)
	}
	if entry.CreatedAt.IsZero() {
		t.Error("entry.CreatedAt is zero, want a timestamp")
	}
}

func TestAddRejectsDuplicateName(t *testing.T) {
	dir := newSpiderProject(t, "blog")
	r, _ := Load(filepath.Join(t.TempDir(), "registry.json"))

	if _, err := r.Add("mysite", dir); err != nil {
		t.Fatalf("first Add() error = %v", err)
	}
	if _, err := r.Add("mysite", dir); err == nil {
		t.Error("second Add() error = nil, want a duplicate-name error")
	}
}

func TestAddRejectsNonSpiderProject(t *testing.T) {
	dir := t.TempDir()
	r, _ := Load(filepath.Join(t.TempDir(), "registry.json"))

	if _, err := r.Add("", dir); err == nil {
		t.Error("Add() error = nil, want an error for a non-Spider directory")
	}
}

func TestAddRejectsMissingPath(t *testing.T) {
	r, _ := Load(filepath.Join(t.TempDir(), "registry.json"))
	if _, err := r.Add("", filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Error("Add() error = nil, want an error for a missing path")
	}
}

func TestRemoveClearsDefault(t *testing.T) {
	dir := newSpiderProject(t, "blog")
	r, _ := Load(filepath.Join(t.TempDir(), "registry.json"))
	if _, err := r.Add("blog", dir); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := r.Use("blog"); err != nil {
		t.Fatalf("Use() error = %v", err)
	}

	if err := r.Remove("blog"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, ok := r.Default(); ok {
		t.Error("Default() ok = true after removing the default site, want false")
	}
}

func TestRemoveUnknownNameErrors(t *testing.T) {
	r, _ := Load(filepath.Join(t.TempDir(), "registry.json"))
	if err := r.Remove("ghost"); err == nil {
		t.Error("Remove() error = nil, want an error for an unregistered name")
	}
}

func TestUseUnknownNameErrors(t *testing.T) {
	r, _ := Load(filepath.Join(t.TempDir(), "registry.json"))
	if err := r.Use("ghost"); err == nil {
		t.Error("Use() error = nil, want an error for an unregistered name")
	}
}

func TestListIsSortedByName(t *testing.T) {
	r, _ := Load(filepath.Join(t.TempDir(), "registry.json"))
	for _, name := range []string{"zebra", "apple", "mango"} {
		if _, err := r.Add(name, newSpiderProject(t, name)); err != nil {
			t.Fatalf("Add(%s) error = %v", name, err)
		}
	}

	got := r.List()
	if len(got) != 3 {
		t.Fatalf("len(List()) = %d, want 3", len(got))
	}
	want := []string{"apple", "mango", "zebra"}
	for i, e := range got {
		if e.Name != want[i] {
			t.Errorf("List()[%d].Name = %q, want %q", i, e.Name, want[i])
		}
	}
}

func TestResolveSiteExplicitNameTakesPrecedence(t *testing.T) {
	r, _ := Load(filepath.Join(t.TempDir(), "registry.json"))
	dirA := newSpiderProject(t, "a")
	dirB := newSpiderProject(t, "b")
	mustAdd(t, r, "a", dirA)
	mustAdd(t, r, "b", dirB)
	if err := r.Use("a"); err != nil {
		t.Fatalf("Use() error = %v", err)
	}

	got, err := r.ResolveSite("b", dirA)
	if err != nil {
		t.Fatalf("ResolveSite() error = %v", err)
	}
	if got.Name != "b" {
		t.Errorf("ResolveSite() = %q, want %q", got.Name, "b")
	}
}

func TestResolveSiteFallsBackToCwd(t *testing.T) {
	r, _ := Load(filepath.Join(t.TempDir(), "registry.json"))
	dirA := newSpiderProject(t, "a")
	mustAdd(t, r, "a", dirA)

	got, err := r.ResolveSite("", dirA)
	if err != nil {
		t.Fatalf("ResolveSite() error = %v", err)
	}
	if got.Name != "a" {
		t.Errorf("ResolveSite() = %q, want %q", got.Name, "a")
	}
}

func TestResolveSiteFallsBackToDefault(t *testing.T) {
	r, _ := Load(filepath.Join(t.TempDir(), "registry.json"))
	dirA := newSpiderProject(t, "a")
	mustAdd(t, r, "a", dirA)
	if err := r.Use("a"); err != nil {
		t.Fatalf("Use() error = %v", err)
	}

	got, err := r.ResolveSite("", t.TempDir())
	if err != nil {
		t.Fatalf("ResolveSite() error = %v", err)
	}
	if got.Name != "a" {
		t.Errorf("ResolveSite() = %q, want %q", got.Name, "a")
	}
}

func TestResolveSiteErrorsWhenNothingApplies(t *testing.T) {
	r, _ := Load(filepath.Join(t.TempDir(), "registry.json"))
	if _, err := r.ResolveSite("", t.TempDir()); err == nil {
		t.Error("ResolveSite() error = nil, want an error when no rule applies")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", "registry.json")
	r, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	dir := newSpiderProject(t, "blog")
	if _, err := r.Add("blog", dir); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := r.Use("blog"); err != nil {
		t.Fatalf("Use() error = %v", err)
	}
	if err := r.Save(path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() after Save() error = %v", err)
	}
	entries := reloaded.List()
	if len(entries) != 1 || entries[0].Name != "blog" {
		t.Fatalf("List() after reload = %+v, want a single %q entry", entries, "blog")
	}
	if _, ok := reloaded.Default(); !ok {
		t.Error("Default() ok = false after reload, want true")
	}
}

func TestSaveWritesOwnerOnlyPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningful on Windows")
	}

	path := filepath.Join(t.TempDir(), "config", "registry.json")
	r, _ := Load(path)
	if err := r.Save(path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", path, err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("registry file mode = %o, want %o", perm, 0o600)
	}
}

func TestLoadMissingFileYieldsEmptyRegistry(t *testing.T) {
	r, err := Load(filepath.Join(t.TempDir(), "registry.json"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(r.List()) != 0 {
		t.Errorf("List() = %v, want an empty registry", r.List())
	}
}

func TestFindByPath(t *testing.T) {
	r, _ := Load(filepath.Join(t.TempDir(), "registry.json"))
	dir := newSpiderProject(t, "blog")
	mustAdd(t, r, "blog", dir)

	if _, ok := r.FindByPath(dir); !ok {
		t.Errorf("FindByPath(%s) ok = false, want true", dir)
	}
	if _, ok := r.FindByPath(t.TempDir()); ok {
		t.Error("FindByPath() ok = true for an unregistered path, want false")
	}
}

func TestStaleReportsWhenPathNoLongerLooksLikeSpider(t *testing.T) {
	dir := newSpiderProject(t, "blog")
	r, _ := Load(filepath.Join(t.TempDir(), "registry.json"))
	entry := mustAdd(t, r, "blog", dir)

	if Stale(entry) {
		t.Error("Stale() = true for a freshly registered project, want false")
	}

	if err := os.Remove(filepath.Join(dir, "bootstrap.php")); err != nil {
		t.Fatalf("Remove(bootstrap.php) error = %v", err)
	}
	if !Stale(entry) {
		t.Error("Stale() = false after removing bootstrap.php, want true")
	}
}

func mustAdd(t *testing.T, r *Registry, name, path string) Entry {
	t.Helper()
	entry, err := r.Add(name, path)
	if err != nil {
		t.Fatalf("Add(%s) error = %v", name, err)
	}
	return entry
}
