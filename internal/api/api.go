package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/cybernetix-lab/graphify-lens/internal/config"
	"github.com/cybernetix-lab/graphify-lens/internal/git"
	"github.com/cybernetix-lab/graphify-lens/internal/graph"
	"github.com/cybernetix-lab/graphify-lens/internal/quality"
	"github.com/cybernetix-lab/graphify-lens/internal/scheduler"
)

type Handler struct {
	scheduler  *scheduler.Scheduler
	cfg        *config.Config
	configPath string
}

func NewHandler(s *scheduler.Scheduler, cfg *config.Config, configPath string) *Handler {
	return &Handler{
		scheduler:  s,
		cfg:        cfg,
		configPath: configPath,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/graph", h.handleGraph)
	mux.HandleFunc("/api/graph/summary", h.handleGraphSummary)
	mux.HandleFunc("/api/graph/query", h.handleGraphQuery)
	mux.HandleFunc("/api/graph/snapshot", h.handleGraphSnapshot)
	mux.HandleFunc("/api/stats", h.handleStats)
	mux.HandleFunc("/api/quality/current", h.handleQualityCurrent)
	mux.HandleFunc("/api/quality/history", h.handleQualityHistory)
	mux.HandleFunc("/api/status", h.handleStatus)
	mux.HandleFunc("/api/cycle/run", h.handleRunCycle)
	mux.HandleFunc("/api/commits", h.handleCommits)
	mux.HandleFunc("/api/commits/log", h.handleCommitLog)
	mux.HandleFunc("/api/config", h.handleConfig)
	mux.HandleFunc("/api/config/workdirs", h.handleWorkDirs)
}

func (h *Handler) resolveWorkDir(r *http.Request) string {
	if wd := r.URL.Query().Get("work_dir"); wd != "" {
		return config.ExpandPath(wd)
	}
	if len(h.cfg.WorkDirs) > 0 {
		return h.cfg.WorkDirs[0]
	}
	return ""
}

func (h *Handler) handleGraph(w http.ResponseWriter, r *http.Request) {
	workDir := h.resolveWorkDir(r)
	summary, err := graph.ParseGraphifyOutSummary(workDir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if summary.TotalNodes > graph.MaxLimit {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"total_nodes": summary.TotalNodes,
			"total_edges": summary.TotalEdges,
			"file_types":  summary.FileTypes,
			"too_large":   true,
			"message":     "Graph too large for full load. Use /api/graph/query with search terms.",
		})
		return
	}

	g, err := graph.ParseGraphifyOut(workDir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (h *Handler) handleGraphSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := graph.ParseGraphifyOutSummary(h.resolveWorkDir(r))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) handleGraphQuery(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := q.Get("q")
	fileType := q.Get("type")

	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = graph.DefaultLimit
	}
	offset, _ := strconv.Atoi(q.Get("offset"))

	gq := graph.GraphQuery{
		Query:    query,
		FileType: fileType,
		Limit:    limit,
		Offset:   offset,
	}

	result, err := graph.ParseGraphifyOutQuery(h.resolveWorkDir(r), gq)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleGraphSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := graph.LoadSnapshot(h.resolveWorkDir(r))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	g, err := graph.ParseGraphifyOut(h.resolveWorkDir(r))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	stats := g.Stats()
	writeJSON(w, http.StatusOK, stats)
}

func (h *Handler) handleQualityCurrent(w http.ResponseWriter, r *http.Request) {
	workDir := h.resolveWorkDir(r)

	result := h.scheduler.LastResult(workDir)
	if result != nil && result.Assessment != nil {
		writeJSON(w, http.StatusOK, result.Assessment)
		return
	}

	g, err := graph.ParseGraphifyOut(workDir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	a, err := quality.Assess(g, h.cfg.QualityHistory)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *Handler) handleQualityHistory(w http.ResponseWriter, r *http.Request) {
	history, err := quality.LoadHistory(h.cfg.QualityHistory)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if history == nil {
		history = []quality.Assessment{}
	}
	writeJSON(w, http.StatusOK, history)
}

func (h *Handler) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := h.scheduler.GetStatus()
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) handleRunCycle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	results := h.scheduler.RunNow()
	writeJSON(w, http.StatusOK, results)
}

func (h *Handler) handleCommits(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results": h.scheduler.AllResults(),
	})
}

type CommitLogEntry struct {
	WorkDir    string           `json:"work_dir"`
	Commits    []git.CommitInfo `json:"commits"`
	TotalCount int              `json:"total_count"`
	Error      string           `json:"error,omitempty"`
}

func (h *Handler) handleCommitLog(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}

	workDir := r.URL.Query().Get("work_dir")

	var entries []CommitLogEntry

	if workDir != "" {
		entry := fetchCommitLog(config.ExpandPath(workDir), limit)
		entries = append(entries, entry)
	} else {
		for _, wd := range h.cfg.WorkDirs {
			entries = append(entries, fetchCommitLog(wd, limit))
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"entries": entries,
	})
}

func fetchCommitLog(workDir string, limit int) CommitLogEntry {
	entry := CommitLogEntry{WorkDir: workDir}
	mgr := git.NewManager(workDir, "", "")
	if !mgr.IsRepo() {
		entry.Error = "not a git repository"
		return entry
	}
	commits, err := mgr.Log(limit)
	if err != nil {
		entry.Error = err.Error()
		return entry
	}
	entry.Commits = commits
	entry.TotalCount = len(commits)
	return entry
}

func (h *Handler) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		resp := map[string]interface{}{
			"work_dirs":       h.cfg.WorkDirs,
			"port":            h.cfg.Port,
			"git_auto_commit": h.cfg.GitAutoCommit,
			"commit_interval": h.cfg.CommitInterval.String(),
			"commit_message":  h.cfg.CommitMessage,
			"quality_history": h.cfg.QualityHistory,
			"data_dir":        h.cfg.DataDir,
			"author_name":     h.cfg.AuthorName,
			"author_email":    h.cfg.AuthorEmail,
		}
		writeJSON(w, http.StatusOK, resp)

	case http.MethodPost:
		var req struct {
			WorkDirs       []string `json:"work_dirs"`
			Port           int      `json:"port"`
			GitAutoCommit  *bool    `json:"git_auto_commit"`
			CommitInterval string   `json:"commit_interval"`
			CommitMessage  string   `json:"commit_message"`
			QualityHistory string   `json:"quality_history"`
			DataDir        string   `json:"data_dir"`
			AuthorName     string   `json:"author_name"`
			AuthorEmail    string   `json:"author_email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if req.WorkDirs != nil {
			expanded := make([]string, len(req.WorkDirs))
			for i, d := range req.WorkDirs {
				expanded[i] = config.ExpandPath(d)
			}
			h.cfg.WorkDirs = expanded
		}
		if req.Port > 0 {
			h.cfg.Port = req.Port
		}
		if req.GitAutoCommit != nil {
			h.cfg.GitAutoCommit = *req.GitAutoCommit
		}
		if req.CommitInterval != "" {
			if d, err := time.ParseDuration(req.CommitInterval); err == nil {
				h.cfg.CommitInterval = d
			}
		}
		if req.CommitMessage != "" {
			h.cfg.CommitMessage = req.CommitMessage
		}
		if req.QualityHistory != "" {
			h.cfg.QualityHistory = config.ExpandPath(req.QualityHistory)
		}
		if req.DataDir != "" {
			h.cfg.DataDir = config.ExpandPath(req.DataDir)
		}
		if req.AuthorName != "" {
			h.cfg.AuthorName = req.AuthorName
		}
		if req.AuthorEmail != "" {
			h.cfg.AuthorEmail = req.AuthorEmail
		}

		h.scheduler.UpdateConfig(h.cfg)
		h.saveConfig()

		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET or POST required"})
	}
}

func (h *Handler) handleWorkDirs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"work_dirs": h.cfg.WorkDirs,
		})

	case http.MethodPost:
		var req struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if req.Path == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path is required"})
			return
		}

		expanded := config.ExpandPath(req.Path)

		for _, d := range h.cfg.WorkDirs {
			if d == expanded {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "path already exists"})
				return
			}
		}
		h.cfg.WorkDirs = append(h.cfg.WorkDirs, expanded)

		h.scheduler.UpdateConfig(h.cfg)
		h.saveConfig()

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":    "ok",
			"work_dirs": h.cfg.WorkDirs,
		})

	case http.MethodDelete:
		path := r.URL.Query().Get("path")
		if path == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path query param required"})
			return
		}

		expanded := config.ExpandPath(path)

		var newDirs []string
		for _, d := range h.cfg.WorkDirs {
			if d != expanded {
				newDirs = append(newDirs, d)
			}
		}
		h.cfg.WorkDirs = newDirs

		h.scheduler.UpdateConfig(h.cfg)
		h.saveConfig()

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":    "ok",
			"work_dirs": h.cfg.WorkDirs,
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET, POST or DELETE required"})
	}
}

func (h *Handler) saveConfig() {
	if h.configPath != "" {
		if err := h.cfg.Save(h.configPath); err != nil {
			log.Printf("[api] save config error: %v", err)
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
