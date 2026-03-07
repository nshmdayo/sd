package stack

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Stack is a file-backed LIFO of directory paths.
type Stack struct {
	entries []string
}

// Load reads the stack file. If the file doesn't exist an empty Stack is
// returned without error.
func Load(path string) (*Stack, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Stack{}, nil
		}
		return nil, fmt.Errorf("read stack: %w", err)
	}
	var entries []string
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if line != "" {
			entries = append(entries, line)
		}
	}
	return &Stack{entries: entries}, nil
}

// Save writes the stack to path atomically.
func (s *Stack) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content := strings.Join(s.entries, "\n")
	if len(s.entries) > 0 {
		content += "\n"
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Push appends dirPath to the top of the stack.
func (s *Stack) Push(dirPath string) error {
	s.entries = append(s.entries, dirPath)
	return nil
}

// Pop removes and returns the top entry.
func (s *Stack) Pop() (string, error) {
	if len(s.entries) == 0 {
		return "", fmt.Errorf("directory stack is empty")
	}
	top := s.entries[len(s.entries)-1]
	s.entries = s.entries[:len(s.entries)-1]
	return top, nil
}

// List returns all entries (bottom to top).
func (s *Stack) List() []string {
	return s.entries
}
