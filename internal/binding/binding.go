// Package binding manages the per-directory .logify TOML file that ties a
// working directory to one Coolify project. Stores reference only — no secrets.
package binding

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const FileName = ".logify"

// File represents the on-disk shape of .logify. One project per directory.
type File struct {
	Project   string `toml:"project"    json:"project"`
	ProjectID string `toml:"project_id" json:"project_id"`
}

// Find walks up from startDir looking for the first .logify file.
// Returns ("", nil) if not found.
func Find(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, FileName)
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

// Load reads + parses the file. If path is "" returns an empty File.
func Load(path string) (*File, error) {
	out := &File{}
	if path == "" {
		return out, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if _, err := toml.Decode(string(b), out); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return out, nil
}

// Save writes the File atomically.
func Save(path string, f *File) error {
	tmp := path + ".tmp"
	w, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := toml.NewEncoder(w).Encode(f); err != nil {
		_ = w.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := w.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

// IsBound reports whether the file has a usable binding.
func (f *File) IsBound() bool {
	return f != nil && f.Project != ""
}
