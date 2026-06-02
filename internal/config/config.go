package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	WorkDirs       []string      `json:"work_dirs"`
	Port           int           `json:"port"`
	GitAutoCommit  bool          `json:"git_auto_commit"`
	CommitInterval time.Duration `json:"commit_interval"`
	CommitMessage  string        `json:"commit_message"`
	QualityHistory string        `json:"quality_history"`
	DataDir        string        `json:"data_dir"`
	AuthorName     string        `json:"author_name"`
	AuthorEmail    string        `json:"author_email"`
}

func CanonicalPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".graphify-lens.json")
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		WorkDirs:       []string{},
		Port:           8080,
		GitAutoCommit:  true,
		CommitInterval: 30 * time.Minute,
		CommitMessage:  "auto: graphify-lens knowledge base snapshot",
		QualityHistory: filepath.Join(home, ".graphify-lens", "quality_history"),
		DataDir:        filepath.Join(home, ".graphify-lens"),
		AuthorName:     "Graphify Lens Bot",
		AuthorEmail:    "graphify-lens-bot@teambuddy.local",
	}
}

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func Load(path string) (*Config, error) {
	canonical := CanonicalPath()

	if path == "" {
		if _, err := os.Stat(canonical); err == nil {
			return loadFile(canonical)
		}
		cfg := DefaultConfig()
		if err := cfg.Save(canonical); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	return loadFile(path)
}

func loadFile(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	type rawConfig struct {
		WorkDir        string   `json:"work_dir"`
		WorkDirs       []string `json:"work_dirs"`
		CurrentWorkDir string   `json:"current_work_dir"`
		Port           int      `json:"port"`
		GitAutoCommit  *bool    `json:"git_auto_commit"`
		CommitInterval string   `json:"commit_interval"`
		CommitMessage  string   `json:"commit_message"`
		QualityHistory string   `json:"quality_history"`
		DataDir        string   `json:"data_dir"`
		AuthorName     string   `json:"author_name"`
		AuthorEmail    string   `json:"author_email"`
	}

	var raw rawConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	if raw.WorkDir != "" && len(raw.WorkDirs) == 0 {
		raw.WorkDirs = []string{raw.WorkDir}
	}

	if len(raw.WorkDirs) > 0 {
		cfg.WorkDirs = make([]string, len(raw.WorkDirs))
		for i, d := range raw.WorkDirs {
			cfg.WorkDirs[i] = ExpandPath(d)
		}
	}

	if raw.Port > 0 {
		cfg.Port = raw.Port
	}
	if raw.GitAutoCommit != nil {
		cfg.GitAutoCommit = *raw.GitAutoCommit
	}
	if raw.CommitInterval != "" {
		d, err := time.ParseDuration(raw.CommitInterval)
		if err == nil {
			cfg.CommitInterval = d
		}
	}
	if raw.CommitMessage != "" {
		cfg.CommitMessage = raw.CommitMessage
	}
	if raw.QualityHistory != "" {
		cfg.QualityHistory = ExpandPath(raw.QualityHistory)
	}
	if raw.DataDir != "" {
		cfg.DataDir = ExpandPath(raw.DataDir)
	}
	if raw.AuthorName != "" {
		cfg.AuthorName = raw.AuthorName
	}
	if raw.AuthorEmail != "" {
		cfg.AuthorEmail = raw.AuthorEmail
	}

	return cfg, nil
}

type configForJSON struct {
	WorkDirs       []string `json:"work_dirs"`
	Port           int      `json:"port"`
	GitAutoCommit  bool     `json:"git_auto_commit"`
	CommitInterval string   `json:"commit_interval"`
	CommitMessage  string   `json:"commit_message"`
	QualityHistory string   `json:"quality_history"`
	DataDir        string   `json:"data_dir"`
	AuthorName     string   `json:"author_name"`
	AuthorEmail    string   `json:"author_email"`
}

func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	raw := configForJSON{
		WorkDirs:       c.WorkDirs,
		Port:           c.Port,
		GitAutoCommit:  c.GitAutoCommit,
		CommitInterval: c.CommitInterval.String(),
		CommitMessage:  c.CommitMessage,
		QualityHistory: c.QualityHistory,
		DataDir:        c.DataDir,
		AuthorName:     c.AuthorName,
		AuthorEmail:    c.AuthorEmail,
	}

	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
