package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/cybernetix-lab/graphify-lens/internal/config"
	"github.com/cybernetix-lab/graphify-lens/internal/graph"
	"github.com/cybernetix-lab/graphify-lens/internal/quality"
	"github.com/cybernetix-lab/graphify-lens/internal/scheduler"
)

type Handler struct {
	scheduler   *scheduler.Scheduler
	cfg         *config.Config
	configPath  string
	workDir     string
	qualityDir  string
	mu          sync.RWMutex
}

func NewHandler(s *scheduler.Scheduler, cfg *config.Config, configPath string) *Handler {
	return &Handler{
		scheduler:  s,
		cfg:        cfg,
		configPath: configPath,
		workDir:    cfg.WorkDir,
		qualityDir: cfg.QualityHistory,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/graph", h.handleGraph)
	mux.HandleFunc("/api/graph/summary", h.handleGraphSummary)
	mux.HandleFunc("/api/graph/query", h.handleGraphQuery)
	mux.HandleFunc("/api/stats", h.handleStats)
	mux.HandleFunc("/api/quality/current", h.handleQualityCurrent)
	mux.HandleFunc("/api/quality/history", h.handleQualityHistory)
	mux.HandleFunc("/api/status", h.handleStatus)
	mux.HandleFunc("/api/cycle/run", h.handleRunCycle)
	mux.HandleFunc("/api/commits", h.handleCommits)
	mux.HandleFunc("/api/config", h.handleConfig)
}

func (h *Handler) getWorkDir() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.workDir
}

func (h *Handler) getQualityDir() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.qualityDir
}

func (h *Handler) handleGraph(w http.ResponseWriter, r *http.Request) {
	workDir := h.getWorkDir()
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
	summary, err := graph.ParseGraphifyOutSummary(h.getWorkDir())
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

	result, err := graph.ParseGraphifyOutQuery(h.getWorkDir(), gq)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	g, err := graph.ParseGraphifyOut(h.getWorkDir())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	stats := g.Stats()
	writeJSON(w, http.StatusOK, stats)
}

func (h *Handler) handleQualityCurrent(w http.ResponseWriter, r *http.Request) {
	result := h.scheduler.LastResult()
	if result != nil && result.Assessment != nil {
		writeJSON(w, http.StatusOK, result.Assessment)
		return
	}

	g, err := graph.ParseGraphifyOut(h.getWorkDir())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	a, err := quality.Assess(g, h.getQualityDir())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *Handler) handleQualityHistory(w http.ResponseWriter, r *http.Request) {
	history, err := quality.LoadHistory(h.getQualityDir())
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
	result := h.scheduler.RunNow()
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleCommits(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"last_cycle": h.scheduler.LastResult(),
	})
}

func (h *Handler) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.mu.RLock()
		resp := map[string]interface{}{
			"work_dir":        h.workDir,
			"port":            h.cfg.Port,
			"git_auto_commit": h.cfg.GitAutoCommit,
			"commit_interval": h.cfg.CommitInterval.String(),
			"commit_message":  h.cfg.CommitMessage,
			"quality_history": h.qualityDir,
			"data_dir":        h.cfg.DataDir,
			"author_name":     h.cfg.AuthorName,
			"author_email":    h.cfg.AuthorEmail,
		}
		h.mu.RUnlock()
		writeJSON(w, http.StatusOK, resp)

	case http.MethodPost:
		var req struct {
			WorkDir        string `json:"work_dir"`
			Port           int    `json:"port"`
			GitAutoCommit  *bool  `json:"git_auto_commit"`
			CommitInterval string `json:"commit_interval"`
			CommitMessage  string `json:"commit_message"`
			QualityHistory string `json:"quality_history"`
			DataDir        string `json:"data_dir"`
			AuthorName     string `json:"author_name"`
			AuthorEmail    string `json:"author_email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		h.mu.Lock()
		if req.WorkDir != "" {
			h.workDir = req.WorkDir
			h.cfg.WorkDir = req.WorkDir
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
			h.qualityDir = req.QualityHistory
			h.cfg.QualityHistory = req.QualityHistory
		}
		if req.DataDir != "" {
			h.cfg.DataDir = req.DataDir
		}
		if req.AuthorName != "" {
			h.cfg.AuthorName = req.AuthorName
		}
		if req.AuthorEmail != "" {
			h.cfg.AuthorEmail = req.AuthorEmail
		}
		h.mu.Unlock()

		h.scheduler.UpdateConfig(h.cfg)

		if h.configPath != "" {
			if err := h.cfg.Save(h.configPath); err != nil {
				log.Printf("[api] save config error: %v", err)
			}
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET or POST required"})
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
