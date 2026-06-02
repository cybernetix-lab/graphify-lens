package api

import (
	"encoding/json"
	"net/http"

	"github.com/cybernetix-lab/graphify-lens/internal/graph"
	"github.com/cybernetix-lab/graphify-lens/internal/quality"
	"github.com/cybernetix-lab/graphify-lens/internal/scheduler"
)

type Handler struct {
	scheduler *scheduler.Scheduler
	workDir   string
	qualityDir string
}

func NewHandler(s *scheduler.Scheduler, workDir, qualityDir string) *Handler {
	return &Handler{
		scheduler:  s,
		workDir:    workDir,
		qualityDir: qualityDir,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/graph", h.handleGraph)
	mux.HandleFunc("/api/stats", h.handleStats)
	mux.HandleFunc("/api/quality/current", h.handleQualityCurrent)
	mux.HandleFunc("/api/quality/history", h.handleQualityHistory)
	mux.HandleFunc("/api/status", h.handleStatus)
	mux.HandleFunc("/api/cycle/run", h.handleRunCycle)
	mux.HandleFunc("/api/commits", h.handleCommits)
}

func (h *Handler) handleGraph(w http.ResponseWriter, r *http.Request) {
	g, err := graph.ParseDir(h.workDir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	g, err := graph.ParseDir(h.workDir)
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

	g, err := graph.ParseDir(h.workDir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	a, err := quality.Assess(g, h.qualityDir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *Handler) handleQualityHistory(w http.ResponseWriter, r *http.Request) {
	history, err := quality.LoadHistory(h.qualityDir)
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

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}


