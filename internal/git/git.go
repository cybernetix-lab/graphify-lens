package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Manager struct {
	WorkDir    string
	AuthorName string
	AuthorEmail string
}

func NewManager(workDir, authorName, authorEmail string) *Manager {
	return &Manager{
		WorkDir:     workDir,
		AuthorName:  authorName,
		AuthorEmail: authorEmail,
	}
}

func (m *Manager) IsRepo() bool {
	cmd := exec.Command("git", "-C", m.WorkDir, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

func (m *Manager) Init() error {
	if m.IsRepo() {
		return nil
	}
	cmd := exec.Command("git", "-C", m.WorkDir, "init")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git init failed: %s", stderr.String())
	}
	return nil
}

func (m *Manager) HasChanges() (bool, error) {
	cmd := exec.Command("git", "-C", m.WorkDir, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(out))) > 0, nil
}

func (m *Manager) AddAll() error {
	cmd := exec.Command("git", "-C", m.WorkDir, "add", "-A")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %s", stderr.String())
	}
	return nil
}

func (m *Manager) Commit(message string) error {
	cmd := exec.Command("git", "-C", m.WorkDir,
		"-c", fmt.Sprintf("user.name=%s", m.AuthorName),
		"-c", fmt.Sprintf("user.email=%s", m.AuthorEmail),
		"commit", "-m", message,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit failed: %s", stderr.String())
	}
	return nil
}

func (m *Manager) Log(n int) ([]CommitInfo, error) {
	cmd := exec.Command("git", "-C", m.WorkDir, "log",
		"--format=%H|%s|%ai|%an",
		"-n", fmt.Sprintf("%d", n),
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var commits []CommitInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		t, _ := time.Parse("2006-01-02 15:04:05 -0700", parts[2])
		commits = append(commits, CommitInfo{
			Hash:    parts[0],
			Message: parts[1],
			Time:    t,
			Author:  parts[3],
		})
	}
	return commits, nil
}

type CommitInfo struct {
	Hash    string    `json:"hash"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
	Author  string    `json:"author"`
}

func (m *Manager) AutoCommit(message string) (*CommitResult, error) {
	result := &CommitResult{
		Time: time.Now(),
	}

	if err := m.Init(); err != nil {
		result.Error = err.Error()
		return result, err
	}

	hasChanges, err := m.HasChanges()
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	if !hasChanges {
		result.Skipped = true
		result.Message = "no changes to commit"
		return result, nil
	}

	if err := m.AddAll(); err != nil {
		result.Error = err.Error()
		return result, err
	}

	if err := m.Commit(message); err != nil {
		result.Error = err.Error()
		return result, err
	}

	result.Committed = true
	result.Message = "commit successful"
	return result, nil
}

type CommitResult struct {
	Time      time.Time `json:"time"`
	Committed bool      `json:"committed"`
	Skipped   bool      `json:"skipped"`
	Message   string    `json:"message"`
	Error     string    `json:"error,omitempty"`
}
