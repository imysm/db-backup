package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSave_FilePermission(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := DefaultConfig()
	cfg.Tasks = nil // no tasks needed for this test

	err := Save(cfg, path)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("config file permission = %o, want 0600", perm)
	}
}
