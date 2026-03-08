package state

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteAtomic writes data to path atomically using a temp file and rename.
// This prevents corruption if two processes write simultaneously.
func WriteAtomic(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp) // clean up on failure
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}
