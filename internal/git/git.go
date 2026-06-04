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
		"--format=%H%x00%s%x00%ai%x00%an",
		"--shortstat",
		"-n", fmt.Sprintf("%d", n),
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var commits []CommitInfo
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x00", 4)
		if len(parts) < 4 {
			continue
		}
		t, _ := time.Parse("2006-01-02 15:04:05 -0700", parts[2])
		c := CommitInfo{
			Hash:    parts[0],
			Message: parts[1],
			Time:    t,
			Author:  parts[3],
		}

		// next non-empty line is the --shortstat line
		for i+1 < len(lines) {
			i++
			statLine := strings.TrimSpace(lines[i])
			if statLine == "" {
				continue
			}
			c.DiffStat = parseShortStat(statLine)
			break
		}

		commits = append(commits, c)
	}
	return commits, nil
}

func parseShortStat(line string) DiffStat {
	// "3 files changed, 15 insertions(+), 2 deletions(-)"
	// or "1 file changed, 5 insertions(+)"
	var ds DiffStat
	parts := strings.Split(line, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.Contains(p, "file") {
			ds.FilesChanged = parseFirstNum(p)
		} else if strings.Contains(p, "insertion") {
			ds.Insertions = parseFirstNum(p)
		} else if strings.Contains(p, "deletion") {
			ds.Deletions = parseFirstNum(p)
		}
	}
	return ds
}

func parseFirstNum(s string) int {
	var n int
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			n = n*10 + int(ch-'0')
		} else if n > 0 {
			break
		}
	}
	return n
}

type CommitInfo struct {
	Hash     string    `json:"hash"`
	Message  string    `json:"message"`
	Time     time.Time `json:"time"`
	Author   string    `json:"author"`
	DiffStat DiffStat   `json:"diff_stat"`
}

type DiffStat struct {
	FilesChanged int `json:"files_changed"`
	Insertions   int `json:"insertions"`
	Deletions    int `json:"deletions"`
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
