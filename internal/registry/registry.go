// Package registry implements spin's persistent record of known Spider
// projects: their filesystem paths and the per-project metadata later
// commands (local domains, web server orchestration, PHP runtime
// management) attach to the same entry.
package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// spiderComposerPackage is the composer.json "name" field Spider's own
// framework package publishes under - the authoritative signal (alongside
// bootstrap.php) that a directory is a Spider project.
const spiderComposerPackage = "spider/framework"

// Entry is one registered Spider project.
type Entry struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Domain     string    `json:"domain,omitempty"`
	PHPVersion string    `json:"php_version,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at,omitempty"`
}

// file is the on-disk JSON shape of the registry.
type file struct {
	Default string           `json:"default,omitempty"`
	Sites   map[string]Entry `json:"sites"`
}

// Registry is the in-memory view of the registry file at a given path,
// loaded via Load and persisted via Save.
type Registry struct {
	data file
}

// Load reads the registry file at path. A missing file is not an error: it
// yields an empty registry, since a fresh machine has none yet.
func Load(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Registry{data: file{Sites: map[string]Entry{}}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var parsed file
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if parsed.Sites == nil {
		parsed.Sites = map[string]Entry{}
	}
	return &Registry{data: parsed}, nil
}

// Save writes the registry to path as indented JSON, atomically (write a
// temporary file in the same directory, then rename over the destination)
// and with owner-only permissions, since registry entries carry filesystem
// paths for the machine they live on.
func (r *Registry) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding registry: %w", err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, ".registry-*.json.tmp")
	if err != nil {
		return fmt.Errorf("creating temporary registry file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op once the rename below succeeds

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", tmpPath, err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return fmt.Errorf("setting permissions on %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("saving %s: %w", path, err)
	}
	return nil
}

// Add registers path under name, deriving name from the directory's base
// name when empty. It fails fast, before touching the registry, when path
// does not exist, is not a directory, does not look like a Spider project,
// or name is already taken.
func (r *Registry) Add(name, path string) (Entry, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return Entry{}, fmt.Errorf("resolving %s: %w", path, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return Entry{}, fmt.Errorf("checking %s: %w", absPath, err)
	}
	if !info.IsDir() {
		return Entry{}, fmt.Errorf("%s is not a directory", absPath)
	}

	if name == "" {
		name = filepath.Base(absPath)
	}
	if strings.ContainsAny(name, `/\`) {
		return Entry{}, fmt.Errorf("site name %q must not contain a path separator", name)
	}
	if _, exists := r.data.Sites[name]; exists {
		return Entry{}, fmt.Errorf("a site named %q is already registered - pick a different --name or run \"spin site remove %s\" first", name, name)
	}

	if !IsSpiderProject(absPath) {
		return Entry{}, fmt.Errorf("%s does not look like a Spider project (no bootstrap.php or matching composer.json)", absPath)
	}

	entry := Entry{
		Name:      name,
		Path:      absPath,
		CreatedAt: time.Now().UTC(),
	}
	r.data.Sites[name] = entry
	return entry, nil
}

// Remove deletes the entry named name, clearing it as the default first if
// it was selected via Use.
func (r *Registry) Remove(name string) error {
	if _, exists := r.data.Sites[name]; !exists {
		return fmt.Errorf("no site named %q is registered", name)
	}
	delete(r.data.Sites, name)
	if r.data.Default == name {
		r.data.Default = ""
	}
	return nil
}

// SetDomain sets the local domain assigned to the site named name.
func (r *Registry) SetDomain(name, domain string) error {
	e, ok := r.data.Sites[name]
	if !ok {
		return fmt.Errorf("no site named %q is registered", name)
	}
	e.Domain = domain
	r.data.Sites[name] = e
	return nil
}

// Use selects name as the default site returned by Default when no other
// resolution rule in ResolveSite applies.
func (r *Registry) Use(name string) error {
	if _, exists := r.data.Sites[name]; !exists {
		return fmt.Errorf("no site named %q is registered", name)
	}
	r.data.Default = name
	return nil
}

// Default returns the site selected via Use, if any.
func (r *Registry) Default() (Entry, bool) {
	if r.data.Default == "" {
		return Entry{}, false
	}
	e, ok := r.data.Sites[r.data.Default]
	return e, ok
}

// List returns every registered entry, sorted by name.
func (r *Registry) List() []Entry {
	entries := make([]Entry, 0, len(r.data.Sites))
	for _, e := range r.data.Sites {
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries
}

// FindByPath returns the entry registered at path, if any. path is compared
// as given; callers pass an already-absolute path.
func (r *Registry) FindByPath(path string) (Entry, bool) {
	for _, e := range r.data.Sites {
		if e.Path == path {
			return e, true
		}
	}
	return Entry{}, false
}

// ResolveSite determines which registered site a command should act on:
// an explicit name (from --site) takes precedence, then the entry whose
// path matches cwd, then the registry's selected default. It fails with a
// clear error rather than guessing when none apply.
func (r *Registry) ResolveSite(explicitName, cwd string) (Entry, error) {
	if explicitName != "" {
		e, ok := r.data.Sites[explicitName]
		if !ok {
			return Entry{}, fmt.Errorf("no site named %q is registered", explicitName)
		}
		return e, nil
	}

	if cwd != "" {
		if absCwd, err := filepath.Abs(cwd); err == nil {
			if e, ok := r.FindByPath(absCwd); ok {
				return e, nil
			}
		}
	}

	if e, ok := r.Default(); ok {
		return e, nil
	}

	return Entry{}, errors.New("no site specified: pass --site <name>, run this from a registered project, or set a default with \"spin site use <name>\"")
}

// IsSpiderProject reports whether path looks like a Spider project root: it
// has a bootstrap.php file, or a composer.json whose "name" field is
// Spider's own framework package.
func IsSpiderProject(path string) bool {
	if _, err := os.Stat(filepath.Join(path, "bootstrap.php")); err == nil {
		return true
	}

	data, err := os.ReadFile(filepath.Join(path, "composer.json"))
	if err != nil {
		return false
	}

	var parsed struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return false
	}
	return parsed.Name == spiderComposerPackage
}

// Stale reports whether e's path no longer looks like a Spider project -
// moved, deleted, or never valid to begin with. Callers surface this rather
// than treating a bad path as a silent error.
func Stale(e Entry) bool {
	return !IsSpiderProject(e.Path)
}
