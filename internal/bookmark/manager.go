package bookmark

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Bookmark represents a saved directory with a user-defined name.
type Bookmark struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
}

// Store holds all bookmarks.
type Store struct {
	Bookmarks []Bookmark `json:"bookmarks"`
}

// Load reads the bookmarks file. If the file doesn't exist an empty Store is
// returned without error.
func Load(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Store{}, nil
		}
		return nil, fmt.Errorf("read bookmarks: %w", err)
	}
	var s Store
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse bookmarks: %w", err)
	}
	return &s, nil
}

// Save atomically writes the store to path.
func (s *Store) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Add inserts or updates a bookmark with the given name and directory path.
func (s *Store) Add(name, dirPath string) error {
	for i, b := range s.Bookmarks {
		if b.Name == name {
			s.Bookmarks[i].Path = dirPath
			return nil
		}
	}
	s.Bookmarks = append(s.Bookmarks, Bookmark{
		Name:      name,
		Path:      dirPath,
		CreatedAt: time.Now().UTC(),
	})
	return nil
}

// Delete removes the bookmark with the given name.
func (s *Store) Delete(name string) error {
	for i, b := range s.Bookmarks {
		if b.Name == name {
			s.Bookmarks = append(s.Bookmarks[:i], s.Bookmarks[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("bookmark %q not found", name)
}

// Find returns the bookmark with the given name or an error.
func (s *Store) Find(name string) (*Bookmark, error) {
	for _, b := range s.Bookmarks {
		if b.Name == name {
			cp := b
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("bookmark %q not found", name)
}

// List returns all bookmarks.
func (s *Store) List() []Bookmark {
	return s.Bookmarks
}

// Names returns just the bookmark names (used for shell completion).
func (s *Store) Names() []string {
	names := make([]string, len(s.Bookmarks))
	for i, b := range s.Bookmarks {
		names[i] = b.Name
	}
	return names
}
