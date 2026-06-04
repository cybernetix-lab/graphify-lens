package scheduler

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cybernetix-lab/graphify-lens/internal/config"
	"github.com/cybernetix-lab/graphify-lens/internal/git"
	"github.com/cybernetix-lab/graphify-lens/internal/graph"
	"github.com/cybernetix-lab/graphify-lens/internal/quality"
)

type Scheduler struct {
	cfg    *config.Config
	ticker *time.Ticker
	stopCh chan struct{}

	mu         sync.RWMutex
	gitMgrs    map[string]*git.Manager
	results    map[string]*CycleResult
}

type CycleResult struct {
	WorkDir    string              `json:"work_dir"`
	Time       time.Time           `json:"time"`
	Commit     *git.CommitResult   `json:"commit"`
	Assessment *quality.Assessment `json:"assessment"`
	GraphStats *graph.GraphStats   `json:"graph_stats"`
}

func New(cfg *config.Config) (*Scheduler, error) {
	s := &Scheduler{
		cfg:     cfg,
		stopCh:  make(chan struct{}),
		gitMgrs: make(map[string]*git.Manager),
		results: make(map[string]*CycleResult),
	}

	for _, wd := range cfg.WorkDirs {
		s.gitMgrs[wd] = git.NewManager(wd, cfg.AuthorName, cfg.AuthorEmail)
	}

	return s, nil
}

func (s *Scheduler) Start() {
	if !s.cfg.GitAutoCommit {
		return
	}

	s.ticker = time.NewTicker(s.cfg.CommitInterval)
	log.Printf("[scheduler] auto-commit started for %d work dirs, interval=%s",
		len(s.cfg.WorkDirs), s.cfg.CommitInterval)

	go s.runLoop()
}

func (s *Scheduler) runLoop() {
	s.runAllCycles()
	for {
		select {
		case <-s.ticker.C:
			s.runAllCycles()
		case <-s.stopCh:
			if s.ticker != nil {
				s.ticker.Stop()
			}
			return
		}
	}
}

func (s *Scheduler) Stop() {
	if s.stopCh != nil {
		close(s.stopCh)
	}
}

func (s *Scheduler) RunNow() map[string]*CycleResult {
	return s.runAllCycles()
}

func (s *Scheduler) runAllCycles() map[string]*CycleResult {
	s.mu.RLock()
	workDirs := make([]string, len(s.cfg.WorkDirs))
	copy(workDirs, s.cfg.WorkDirs)
	qualityHistory := s.cfg.QualityHistory
	s.mu.RUnlock()

	results := make(map[string]*CycleResult)

	for _, wd := range workDirs {
		result := s.runCycle(wd, qualityHistory)
		results[wd] = result

		s.mu.Lock()
		s.results[wd] = result
		s.mu.Unlock()
	}

	return results
}

func (s *Scheduler) runCycle(workDir, qualityHistory string) *CycleResult {
	result := &CycleResult{
		WorkDir: workDir,
		Time:    time.Now(),
	}

	if workDir == "" {
		log.Printf("[scheduler] empty work dir, skipping")
		return result
	}

	g, err := graph.ParseGraphifyOut(workDir)
	if err != nil {
		log.Printf("[scheduler] graph parse error [%s]: %v", workDir, err)
		return result
	}

	stats := g.Stats()
	result.GraphStats = &stats

	snapshot, err := graph.GenerateSnapshot(workDir)
	if err != nil {
		log.Printf("[scheduler] snapshot generation error [%s]: %v", workDir, err)
	} else {
		if err := graph.SaveSnapshot(snapshot, workDir); err != nil {
			log.Printf("[scheduler] snapshot save error [%s]: %v", workDir, err)
		}
	}

	assessment, err := quality.Assess(g, qualityHistory)
	if err != nil {
		log.Printf("[scheduler] quality assess error [%s]: %v", workDir, err)
	} else {
		result.Assessment = assessment
		if err := quality.SaveAssessment(assessment, qualityHistory); err != nil {
			log.Printf("[scheduler] save assessment error [%s]: %v", workDir, err)
		}
	}

	s.mu.RLock()
	gitMgr := s.gitMgrs[workDir]
	prevResult := s.results[workDir]
	s.mu.RUnlock()

	if gitMgr != nil {
		msg := buildCommitMessage(g, assessment, prevResult)
		commitResult, err := gitMgr.AutoCommit(msg)
		if err != nil {
			log.Printf("[scheduler] auto commit error [%s]: %v", workDir, err)
		}
		result.Commit = commitResult
	}

	log.Printf("[scheduler] cycle complete [%s]: committed=%v, score=%.2f",
		workDir,
		func() bool {
			if result.Commit != nil {
				return result.Commit.Committed
			}
			return false
		}(),
		func() float64 {
			if assessment != nil {
				return assessment.TotalScore
			}
			return 0
		}(),
	)

	return result
}

func buildCommitMessage(g *graph.Graph, a *quality.Assessment, prev *CycleResult) string {
	var parts []string

	// node/edge counts
	parts = append(parts, fmt.Sprintf("nodes=%d edges=%d", len(g.Nodes), len(g.Edges)))

	// delta from previous cycle
	if prev != nil && prev.GraphStats != nil {
		nodeDelta := len(g.Nodes) - prev.GraphStats.TotalNodes
		edgeDelta := len(g.Edges) - prev.GraphStats.TotalEdges
		if nodeDelta != 0 || edgeDelta != 0 {
			parts = append(parts, fmt.Sprintf("Δ nodes%+d edges%+d", nodeDelta, edgeDelta))
		}
	}

	// file type summary
	typeCounts := make(map[string]int)
	for _, n := range g.Nodes {
		ft := n.FileType
		if ft == "" {
			ft = "other"
		}
		typeCounts[ft]++
	}
	if len(typeCounts) > 0 {
		type typeKV struct {
			k string
			v int
		}
		var sorted []typeKV
		for k, v := range typeCounts {
			sorted = append(sorted, typeKV{k, v})
		}
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].v > sorted[j].v })
		var typeParts []string
		for _, kv := range sorted {
			typeParts = append(typeParts, fmt.Sprintf("%s:%d", kv.k, kv.v))
		}
		parts = append(parts, strings.Join(typeParts, " "))
	}

	// quality score
	if a != nil {
		parts = append(parts, fmt.Sprintf("score=%.0f", a.TotalScore))
	}

	// top relations
	relCounts := make(map[string]int)
	for _, e := range g.Edges {
		relCounts[e.Relation]++
	}
	if len(relCounts) > 0 {
		type relKV struct {
			k string
			v int
		}
		var sorted []relKV
		for k, v := range relCounts {
			sorted = append(sorted, relKV{k, v})
		}
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].v > sorted[j].v })
		topN := 3
		if len(sorted) < topN {
			topN = len(sorted)
		}
		var relParts []string
		for i := 0; i < topN; i++ {
			relParts = append(relParts, fmt.Sprintf("%s:%d", sorted[i].k, sorted[i].v))
		}
		parts = append(parts, "top: "+strings.Join(relParts, " "))
	}

	return "graphify: " + strings.Join(parts, " | ")
}

func (s *Scheduler) LastResult(workDir string) *CycleResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.results[workDir]
}

func (s *Scheduler) AllResults() map[string]*CycleResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]*CycleResult, len(s.results))
	for k, v := range s.results {
		out[k] = v
	}
	return out
}

func (s *Scheduler) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := map[string]interface{}{
		"auto_commit_enabled": s.cfg.GitAutoCommit,
		"commit_interval":     s.cfg.CommitInterval.String(),
		"work_dirs":           s.cfg.WorkDirs,
		"cron_running":        s.ticker != nil,
	}

	dirStatuses := make([]map[string]interface{}, 0, len(s.cfg.WorkDirs))
	for _, wd := range s.cfg.WorkDirs {
		ds := map[string]interface{}{
			"work_dir": wd,
		}
		if r, ok := s.results[wd]; ok {
			ds["last_cycle"] = r.Time.Format(time.RFC3339)
			if r.Assessment != nil {
				ds["last_score"] = r.Assessment.TotalScore
			}
			if r.Commit != nil {
				ds["last_commit"] = r.Commit.Committed
			}
		}
		dirStatuses = append(dirStatuses, ds)
	}
	status["dir_statuses"] = dirStatuses

	return status
}

func (s *Scheduler) UpdateConfig(cfg *config.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldDirs := make(map[string]bool)
	for _, d := range s.cfg.WorkDirs {
		oldDirs[d] = true
	}

	s.cfg = cfg

	newGitMgrs := make(map[string]*git.Manager)
	for _, wd := range cfg.WorkDirs {
		if mgr, ok := s.gitMgrs[wd]; ok {
			newGitMgrs[wd] = mgr
		} else {
			newGitMgrs[wd] = git.NewManager(wd, cfg.AuthorName, cfg.AuthorEmail)
			log.Printf("[scheduler] new work dir registered: %s", wd)
		}
	}
	s.gitMgrs = newGitMgrs

	for _, wd := range cfg.WorkDirs {
		if !oldDirs[wd] {
			log.Printf("[scheduler] work dir added: %s", wd)
		}
	}

	if s.ticker != nil {
		s.ticker.Stop()
	}
	if cfg.GitAutoCommit {
		s.ticker = time.NewTicker(cfg.CommitInterval)
		log.Printf("[scheduler] auto-commit reconfigured, interval=%s", cfg.CommitInterval)
	} else {
		s.ticker = nil
	}
}
