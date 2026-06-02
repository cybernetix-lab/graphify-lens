package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func skipIfNoGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}
}

func TestNewManager(t *testing.T) {
	skipIfNoGit(t)
	m := NewManager("/tmp", "test", "test@test.com")
	if m.WorkDir != "/tmp" {
		t.Errorf("expected /tmp, got %s", m.WorkDir)
	}
}

func TestInit(t *testing.T) {
	skipIfNoGit(t)
	dir := t.TempDir()
	m := NewManager(dir, "test", "test@test.com")

	if err := m.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	if !m.IsRepo() {
		t.Error("expected repo after init")
	}
}

func TestInit_Idempotent(t *testing.T) {
	skipIfNoGit(t)
	dir := t.TempDir()
	m := NewManager(dir, "test", "test@test.com")

	if err := m.Init(); err != nil {
		t.Fatalf("first init failed: %v", err)
	}
	if err := m.Init(); err != nil {
		t.Fatalf("second init failed: %v", err)
	}
}

func TestHasChanges_EmptyRepo(t *testing.T) {
	skipIfNoGit(t)
	dir := t.TempDir()
	m := NewManager(dir, "test", "test@test.com")
	m.Init()

	hasChanges, err := m.HasChanges()
	if err != nil {
		t.Fatalf("hasChanges failed: %v", err)
	}
	if hasChanges {
		t.Error("expected no changes in empty repo")
	}
}

func TestHasChanges_WithFile(t *testing.T) {
	skipIfNoGit(t)
	dir := t.TempDir()
	m := NewManager(dir, "test", "test@test.com")
	m.Init()

	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0644)

	hasChanges, err := m.HasChanges()
	if err != nil {
		t.Fatalf("hasChanges failed: %v", err)
	}
	if !hasChanges {
		t.Error("expected changes after file creation")
	}
}

func TestAddAndCommit(t *testing.T) {
	skipIfNoGit(t)
	dir := t.TempDir()
	m := NewManager(dir, "Test Author", "test@test.com")
	m.Init()

	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0644)

	if err := m.AddAll(); err != nil {
		t.Fatalf("add failed: %v", err)
	}
	if err := m.Commit("test commit"); err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	hasChanges, err := m.HasChanges()
	if err != nil {
		t.Fatalf("hasChanges failed: %v", err)
	}
	if hasChanges {
		t.Error("expected no changes after commit")
	}
}

func TestAutoCommit_NoChanges(t *testing.T) {
	skipIfNoGit(t)
	dir := t.TempDir()
	m := NewManager(dir, "test", "test@test.com")
	m.Init()

	result, err := m.AutoCommit("test message")
	if err != nil {
		t.Fatalf("autoCommit failed: %v", err)
	}
	if !result.Skipped {
		t.Error("expected skipped when no changes")
	}
	if result.Committed {
		t.Error("expected not committed when no changes")
	}
}

func TestAutoCommit_WithChanges(t *testing.T) {
	skipIfNoGit(t)
	dir := t.TempDir()
	m := NewManager(dir, "Test Author", "test@test.com")
	m.Init()

	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0644)

	result, err := m.AutoCommit("test message")
	if err != nil {
		t.Fatalf("autoCommit failed: %v", err)
	}
	if !result.Committed {
		t.Error("expected committed with changes")
	}
}

func TestLog(t *testing.T) {
	skipIfNoGit(t)
	dir := t.TempDir()
	m := NewManager(dir, "Test Author", "test@test.com")
	m.Init()

	os.WriteFile(filepath.Join(dir, "f1.txt"), []byte("a"), 0644)
	m.AddAll()
	m.Commit("first commit")

	os.WriteFile(filepath.Join(dir, "f2.txt"), []byte("b"), 0644)
	m.AddAll()
	m.Commit("second commit")

	logs, err := m.Log(5)
	if err != nil {
		t.Fatalf("log failed: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(logs))
	}
	if logs[0].Message != "second commit" {
		t.Errorf("expected 'second commit', got '%s'", logs[0].Message)
	}
}

func TestIsRepo_False(t *testing.T) {
	skipIfNoGit(t)
	dir := t.TempDir()
	m := NewManager(dir, "test", "test@test.com")
	if m.IsRepo() {
		t.Error("expected not a repo")
	}
}
