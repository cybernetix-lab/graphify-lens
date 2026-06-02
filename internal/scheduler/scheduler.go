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
	cfg        *config.Config
	gitMgr     *git.Manager
	ticker     *time.Ticker
	stopCh     chan struct{}
	lastResult *CycleResult
	mu         sync.RWMutex
}

type CycleResult struct {
	Time       time.Time           `json:"time"`
	Commit     *git.CommitResult   `json:"commit"`
	Assessment *quality.Assessment `json:"assessment"`
	GraphStats *graph.GraphStats   `json:"graph_stats"`
}

func New(cfg *config.Config) (*Scheduler, error) {
	s := &Scheduler{
		cfg:    cfg,
		gitMgr: git.NewManager(cfg.WorkDir, cfg.AuthorName, cfg.AuthorEmail),
		stopCh: make(chan struct{}),
	}

	return s, nil
}

func (s *Scheduler) Start() {
	if !s.cfg.GitAutoCommit {
		return
	}

	s.ticker = time.NewTicker(s.cfg.CommitInterval)
	log.Printf("[scheduler] auto-commit started, interval=%s", s.cfg.CommitInterval)

	go func() {
		s.runCycle()
		for {
			select {
			case <-s.ticker.C:
				s.runCycle()
			case <-s.stopCh:
				s.ticker.Stop()
				return
			}
		}
	}()
}

func (s *Scheduler) Stop() {
	if s.stopCh != nil {
		close(s.stopCh)
	}
}

func (s *Scheduler) RunNow() *CycleResult {
	return s.runCycle()
}

func (s *Scheduler) runCycle() *CycleResult {
	result := &CycleResult{Time: time.Now()}

	g, err := graph.ParseGraphifyOut(s.cfg.WorkDir)
	if err != nil {
		log.Printf("[scheduler] graph parse error: %v", err)
		return result
	}

	stats := g.Stats()
	result.GraphStats = &stats

	assessment, err := quality.Assess(g, s.cfg.QualityHistory)
	if err != nil {
		log.Printf("[scheduler] quality assess error: %v", err)
	} else {
		result.Assessment = assessment
		if err := quality.SaveAssessment(assessment, s.cfg.QualityHistory); err != nil {
			log.Printf("[scheduler] save assessment error: %v", err)
		}
	}

	commitResult, err := s.gitMgr.AutoCommit(s.cfg.CommitMessage)
	if err != nil {
		log.Printf("[scheduler] auto commit error: %v", err)
	}
	result.Commit = commitResult

	s.mu.Lock()
	s.lastResult = result
	s.mu.Unlock()

	log.Printf("[scheduler] cycle complete: committed=%v, score=%.2f",
		commitResult.Committed,
		func() float64 {
			if assessment != nil {
				return assessment.TotalScore
			}
			return 0
		}(),
	)

	return result
}

func (s *Scheduler) LastResult() *CycleResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastResult
}

func (s *Scheduler) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := map[string]interface{}{
		"auto_commit_enabled": s.cfg.GitAutoCommit,
		"commit_interval":     s.cfg.CommitInterval.String(),
		"work_dir":            s.cfg.WorkDir,
		"cron_running":        s.ticker != nil,
	}

	if s.lastResult != nil {
		status["last_cycle"] = s.lastResult.Time.Format(time.RFC3339)
		if s.lastResult.Assessment != nil {
			status["last_score"] = s.lastResult.Assessment.TotalScore
		}
		if s.lastResult.Commit != nil {
			status["last_commit"] = s.lastResult.Commit.Committed
		}
	}

	return status
}
