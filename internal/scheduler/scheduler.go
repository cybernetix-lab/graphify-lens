package scheduler

import (
	"log"
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
	commitMessage := s.cfg.CommitMessage
	s.mu.RUnlock()

	results := make(map[string]*CycleResult)

	for _, wd := range workDirs {
		result := s.runCycle(wd, qualityHistory, commitMessage)
		results[wd] = result

		s.mu.Lock()
		s.results[wd] = result
		s.mu.Unlock()
	}

	return results
}

func (s *Scheduler) runCycle(workDir, qualityHistory, commitMessage string) *CycleResult {
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
	s.mu.RUnlock()

	if gitMgr != nil {
		commitResult, err := gitMgr.AutoCommit(commitMessage)
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
