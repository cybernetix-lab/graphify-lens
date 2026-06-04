package quality

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/cybernetix-lab/graphify-lens/internal/graph"
)

type Assessment struct {
	Timestamp   time.Time        `json:"timestamp"`
	Coverage    CoverageScore    `json:"coverage"`
	Accuracy    AccuracyScore    `json:"accuracy"`
	Freshness   FreshnessScore   `json:"freshness"`
	Governance  GovernanceScore  `json:"governance"`
	ReuseGrowth ReuseGrowthScore `json:"reuse_growth"`
	TotalScore  float64          `json:"total_score"`
	Version     string           `json:"version"`
}

type CoverageScore struct {
	Score          float64 `json:"score"`
	NodeCount      int     `json:"node_count"`
	EdgeCount      int     `json:"edge_count"`
	FileTypeCount  int     `json:"file_type_count"`
	TypeDiversity  float64 `json:"type_diversity"`
	EdgeDensity    float64 `json:"edge_density"`
}

type AccuracyScore struct {
	Score            float64 `json:"score"`
	ExtractedRate    float64 `json:"extracted_rate"`
	InferredRate     float64 `json:"inferred_rate"`
	AmbiguousRate    float64 `json:"ambiguous_rate"`
	AvgConfidence    float64 `json:"avg_confidence"`
}

type FreshnessScore struct {
	Score              float64 `json:"score"`
	NodesWithTimestamp  int     `json:"nodes_with_timestamp"`
	AvgDaysSinceCapture float64 `json:"avg_days_since_capture"`
	StaleNodeRatio     float64 `json:"stale_node_ratio"`
}

type GovernanceScore struct {
	Score              float64 `json:"score"`
	AuthorCoverage     float64 `json:"author_coverage"`
	ContributorCoverage float64 `json:"contributor_coverage"`
	SourceFileCoverage float64 `json:"source_file_coverage"`
}

type ReuseGrowthScore struct {
	Score          float64 `json:"score"`
	NodeGrowthRate float64 `json:"node_growth_rate"`
	EdgeGrowthRate float64 `json:"edge_growth_rate"`
	KnowledgeROI   float64 `json:"knowledge_roi"`
}

const (
	weightCoverage    = 0.30
	weightAccuracy    = 0.25
	weightFreshness   = 0.20
	weightGovernance  = 0.15
	weightReuseGrowth = 0.10
)

func Assess(g *graph.Graph, historyDir string) (*Assessment, error) {
	a := &Assessment{
		Timestamp: time.Now(),
		Version:   "1.0",
	}

	a.Coverage = assessCoverage(g)
	a.Accuracy = assessAccuracy(g)
	a.Freshness = assessFreshness(g)
	a.Governance = assessGovernance(g)
	a.ReuseGrowth = assessReuseGrowth(g, historyDir)

	a.TotalScore = math.Round(
		(a.Coverage.Score*weightCoverage+
			a.Accuracy.Score*weightAccuracy+
			a.Freshness.Score*weightFreshness+
			a.Governance.Score*weightGovernance+
			a.ReuseGrowth.Score*weightReuseGrowth)*100,
	)

	return a, nil
}

func assessCoverage(g *graph.Graph) CoverageScore {
	cs := CoverageScore{
		NodeCount: len(g.Nodes),
		EdgeCount: len(g.Edges),
	}

	typeSet := make(map[string]bool)
	for _, n := range g.Nodes {
		if n.FileType != "" {
			typeSet[n.FileType] = true
		}
	}
	cs.FileTypeCount = len(typeSet)
	cs.TypeDiversity = math.Min(float64(cs.FileTypeCount)/5.0, 1.0)

	if len(g.Nodes) > 0 {
		cs.EdgeDensity = math.Min(float64(len(g.Edges))/float64(len(g.Nodes)*2), 1.0)
	}

	nodeCountScore := math.Min(float64(cs.NodeCount)/50.0, 1.0)
	cs.Score = nodeCountScore*0.3 + cs.TypeDiversity*0.4 + cs.EdgeDensity*0.3
	return cs
}

func assessAccuracy(g *graph.Graph) AccuracyScore {
	as := AccuracyScore{}

	extracted := 0
	inferred := 0
	ambiguous := 0
	totalConf := 0.0

	for _, e := range g.Edges {
		switch e.Confidence {
		case "EXTRACTED":
			extracted++
		case "INFERRED":
			inferred++
		case "AMBIGUOUS":
			ambiguous++
		}
		totalConf += e.ConfidenceScore
	}

	total := len(g.Edges)
	if total > 0 {
		as.ExtractedRate = float64(extracted) / float64(total)
		as.InferredRate = float64(inferred) / float64(total)
		as.AmbiguousRate = float64(ambiguous) / float64(total)
		as.AvgConfidence = totalConf / float64(total)
	} else {
		as.ExtractedRate = 1.0
		as.AvgConfidence = 1.0
	}

	as.Score = as.ExtractedRate*0.5 + as.AvgConfidence*0.3 + (1.0-as.AmbiguousRate)*0.2
	return as
}

func assessFreshness(g *graph.Graph) FreshnessScore {
	fs := FreshnessScore{}

	now := time.Now()
	withTimestamp := 0
	totalDays := 0.0
	staleCount := 0

	for _, n := range g.Nodes {
		if n.CapturedAt == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, n.CapturedAt)
		if err != nil {
			continue
		}
		days := now.Sub(t).Hours() / 24
		totalDays += days
		withTimestamp++

		if days > 180 {
			staleCount++
		}
	}

	fs.NodesWithTimestamp = withTimestamp
	if withTimestamp > 0 {
		fs.AvgDaysSinceCapture = totalDays / float64(withTimestamp)
	}

	if len(g.Nodes) > 0 {
		fs.StaleNodeRatio = float64(staleCount) / float64(len(g.Nodes))
	}

	freshnessCompliance := 1.0 - fs.StaleNodeRatio
	ageScore := math.Max(0, 1.0-fs.AvgDaysSinceCapture/365.0)
	fs.Score = freshnessCompliance*0.6 + ageScore*0.4
	return fs
}

func assessGovernance(g *graph.Graph) GovernanceScore {
	gs := GovernanceScore{}

	hasAuthor := 0
	hasContributor := 0
	hasSourceFile := 0

	for _, n := range g.Nodes {
		if n.Author != "" {
			hasAuthor++
		}
		if n.Contributor != "" {
			hasContributor++
		}
		if n.SourceFile != "" {
			hasSourceFile++
		}
	}

	total := len(g.Nodes)
	if total > 0 {
		gs.AuthorCoverage = float64(hasAuthor) / float64(total)
		gs.ContributorCoverage = float64(hasContributor) / float64(total)
		gs.SourceFileCoverage = float64(hasSourceFile) / float64(total)
	} else {
		gs.AuthorCoverage = 1.0
		gs.ContributorCoverage = 1.0
		gs.SourceFileCoverage = 1.0
	}

	gs.Score = gs.AuthorCoverage*0.4 + gs.SourceFileCoverage*0.4 + gs.ContributorCoverage*0.2
	return gs
}

func assessReuseGrowth(g *graph.Graph, historyDir string) ReuseGrowthScore {
	rgs := ReuseGrowthScore{}

	prev := loadPreviousAssessment(historyDir)
	if prev != nil {
		prevNodes := float64(prev.Coverage.NodeCount)
		prevEdges := float64(prev.Coverage.EdgeCount)
		currNodes := float64(len(g.Nodes))
		currEdges := float64(len(g.Edges))

		if prevNodes > 0 {
			rgs.NodeGrowthRate = (currNodes - prevNodes) / prevNodes
		}
		if prevEdges > 0 {
			rgs.EdgeGrowthRate = (currEdges - prevEdges) / prevEdges
		}

		if prev.TotalScore > 0 {
			rgs.KnowledgeROI = math.Max(0, (rgs.NodeGrowthRate+rgs.EdgeGrowthRate)/2)
		}
	}

	rgs.Score = math.Min(math.Max(rgs.NodeGrowthRate*5, 0)+math.Max(rgs.EdgeGrowthRate*5, 0)+math.Min(rgs.KnowledgeROI*10, 1.0)*0.3, 1.0)
	return rgs
}

func loadPreviousAssessment(historyDir string) *Assessment {
	entries, err := os.ReadDir(historyDir)
	if err != nil {
		return nil
	}

	var latest *Assessment
	var latestTime time.Time
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(historyDir, entry.Name()))
		if err != nil {
			continue
		}
		var a Assessment
		if err := json.Unmarshal(data, &a); err != nil {
			continue
		}
		if a.Timestamp.After(latestTime) {
			latestTime = a.Timestamp
			latest = &a
		}
	}
	return latest
}

func SaveAssessment(a *Assessment, historyDir string) error {
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return err
	}
	filename := fmt.Sprintf("quality_%s.json", a.Timestamp.Format("20060102_150405"))
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(historyDir, filename), data, 0644)
}

func LoadHistory(historyDir string) ([]Assessment, error) {
	entries, err := os.ReadDir(historyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var assessments []Assessment
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(historyDir, entry.Name()))
		if err != nil {
			continue
		}
		var a Assessment
		if err := json.Unmarshal(data, &a); err != nil {
			continue
		}
		assessments = append(assessments, a)
	}
	return assessments, nil
}
