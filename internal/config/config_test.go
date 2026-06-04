package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
	if cfg.GitAutoCommit != true {
		t.Error("expected git_auto_commit to be true")
	}
	if cfg.CommitInterval != 30*time.Minute {
		t.Errorf("expected 30m interval, got %s", cfg.CommitInterval)
	}
}

func TestLoad_NoFile(t *testing.T) {
	canonical := CanonicalPath()
	os.Remove(canonical)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected default port, got %d", cfg.Port)
	}

	if _, err := os.Stat(canonical); err != nil {
		t.Errorf("expected canonical config to be created at %s: %v", canonical, err)
	}
}

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{"port": 9090, "git_auto_commit": false, "commit_interval": "1h", "work_dirs": ["/tmp/test"]}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Port)
	}
	if cfg.GitAutoCommit != false {
		t.Error("expected git_auto_commit to be false")
	}
	if cfg.CommitInterval != 1*time.Hour {
		t.Errorf("expected 1h interval, got %s", cfg.CommitInterval)
	}
	if len(cfg.WorkDirs) != 1 || cfg.WorkDirs[0] != "/tmp/test" {
		t.Errorf("expected work_dirs [/tmp/test], got %v", cfg.WorkDirs)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{invalid`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoad_BackwardCompatWorkDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{"work_dir": "/tmp/oldstyle"}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.WorkDirs) != 1 || cfg.WorkDirs[0] != "/tmp/oldstyle" {
		t.Errorf("expected work_dirs [/tmp/oldstyle], got %v", cfg.WorkDirs)
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"/tmp/test", "/tmp/test"},
		{"./relative", "./relative"},
		{"~/Projects/monorepo", filepath.Join(home, "Projects/monorepo")},
		{"~", home},
	}

	for _, tt := range tests {
		result := ExpandPath(tt.input)
		if result != tt.expected {
			t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := DefaultConfig()
	cfg.Port = 7070

	if err := cfg.Save(path); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.Port != 7070 {
		t.Errorf("expected port 7070, got %d", loaded.Port)
	}
}
