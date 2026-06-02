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
	Timestamp   time.Time     `json:"timestamp"`
	Coverage    CoverageScore `json:"coverage"`
	Accuracy    AccuracyScore `json:"accuracy"`
	Freshness   FreshnessScore `json:"freshness"`
	Governance  GovernanceScore `json:"governance"`
	ReuseGrowth ReuseGrowthScore `json:"reuse_growth"`
	TotalScore  float64       `json:"total_score"`
	Version     string        `json:"version"`
}

type CoverageScore struct {
	Score            float64 `json:"score"`
	TopicCoverage    float64 `json:"topic_coverage"`
	ScenarioCoverage float64 `json:"scenario_coverage"`
	TypeCoverage     float64 `json:"type_coverage"`
}

type AccuracyScore struct {
	Score              float64 `json:"score"`
	GroundedEdgeRate   float64 `json:"grounded_edge_rate"`
	ConflictExposure   float64 `json:"conflict_exposure"`
	SourceRefCoverage  float64 `json:"source_ref_coverage"`
}

type FreshnessScore struct {
	Score              float64 `json:"score"`
	FreshnessCompliance float64 `json:"freshness_compliance"`
	StalePageRatio     float64 `json:"stale_page_ratio"`
	AvgDaysSinceReview float64 `json:"avg_days_since_review"`
}

type GovernanceScore struct {
	Score                float64 `json:"score"`
	MetadataCompleteness float64 `json:"metadata_completeness"`
	OwnerCoverage        float64 `json:"owner_coverage"`
	PermissionCompliance float64 `json:"permission_compliance"`
}

type ReuseGrowthScore struct {
	Score               float64 `json:"score"`
	ReuseRate           float64 `json:"reuse_rate"`
	ConversionRate      float64 `json:"conversion_rate"`
	KnowledgeROI        float64 `json:"knowledge_roi"`
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
		a.Coverage.Score*weightCoverage*100 +
			a.Accuracy.Score*weightAccuracy*100 +
			a.Freshness.Score*weightFreshness*100 +
			a.Governance.Score*weightGovernance*100 +
			a.ReuseGrowth.Score*weightReuseGrowth*100,
	) / 100

	return a, nil
}

func assessCoverage(g *graph.Graph) CoverageScore {
	cs := CoverageScore{}

	expectedTypes := 6.0
	actualTypes := 0.0
	typeSet := make(map[graph.NodeType]bool)
	for _, n := range g.Nodes {
		if n.Type != "" {
			typeSet[n.Type] = true
		}
	}
	actualTypes = float64(len(typeSet))
	cs.TypeCoverage = math.Min(actualTypes/expectedTypes, 1.0)

	topicCount := 0
	for _, n := range g.Nodes {
		if n.Type == graph.NodeTypeTopic {
			topicCount++
		}
	}
	cs.TopicCoverage = math.Min(float64(topicCount)/5.0, 1.0)

	scenarioSet := make(map[string]bool)
	for _, n := range g.Nodes {
		for _, tag := range n.ScenarioTags {
			scenarioSet[tag] = true
		}
	}
	cs.ScenarioCoverage = math.Min(float64(len(scenarioSet))/3.0, 1.0)

	cs.Score = cs.TypeCoverage*0.4 + cs.TopicCoverage*0.3 + cs.ScenarioCoverage*0.3
	return cs
}

func assessAccuracy(g *graph.Graph) AccuracyScore {
	as := AccuracyScore{}

	groundedEdges := 0
	for _, e := range g.Edges {
		if e.Confidence > 0.5 {
			groundedEdges++
		}
	}
	if len(g.Edges) > 0 {
		as.GroundedEdgeRate = float64(groundedEdges) / float64(len(g.Edges))
	} else {
		as.GroundedEdgeRate = 1.0
	}

	nodesWithRefs := 0
	for _, n := range g.Nodes {
		if len(n.SourceRefs) > 0 {
			nodesWithRefs++
		}
	}
	if len(g.Nodes) > 0 {
		as.SourceRefCoverage = float64(nodesWithRefs) / float64(len(g.Nodes))
	} else {
		as.SourceRefCoverage = 1.0
	}

	as.ConflictExposure = 0.0

	as.Score = as.GroundedEdgeRate*0.5 + as.SourceRefCoverage*0.4 + (1.0-as.ConflictExposure)*0.1
	return as
}

func assessFreshness(g *graph.Graph) FreshnessScore {
	fs := FreshnessScore{}

	now := time.Now()
	staleCount := 0
	totalDays := 0.0
	reviewedCount := 0

	for _, n := range g.Nodes {
		if n.LastReviewedAt == "" {
			staleCount++
			continue
		}
		t, err := time.Parse(time.RFC3339, n.LastReviewedAt)
		if err != nil {
			staleCount++
			continue
		}
		days := now.Sub(t).Hours() / 24
		totalDays += days
		reviewedCount++

		sla := n.FreshnessSLA
		if sla == "" {
			sla = "720h"
		}
		slaDuration, err := time.ParseDuration(sla)
		if err != nil {
			slaDuration = 720 * time.Hour
		}
		if days > slaDuration.Hours()/24 {
			staleCount++
		}
	}

	if len(g.Nodes) > 0 {
		fs.StalePageRatio = float64(staleCount) / float64(len(g.Nodes))
		fs.FreshnessCompliance = 1.0 - fs.StalePageRatio
	} else {
		fs.FreshnessCompliance = 1.0
	}

	if reviewedCount > 0 {
		fs.AvgDaysSinceReview = totalDays / float64(reviewedCount)
	}

	fs.Score = fs.FreshnessCompliance*0.6 + math.Max(0, 1.0-fs.AvgDaysSinceReview/365.0)*0.4
	return fs
}

func assessGovernance(g *graph.Graph) GovernanceScore {
	gs := GovernanceScore{}

	requiredFields := []string{"owner", "classification", "visibility_scope", "status"}
	completeNodes := 0
	hasOwner := 0
	hasPermission := 0

	for _, n := range g.Nodes {
		complete := true
		if n.Owner == "" {
			complete = false
		} else {
			hasOwner++
		}
		if n.Classification == "" {
			complete = false
		}
		if n.Visibility == "" {
			complete = false
		} else {
			hasPermission++
		}
		if n.Status == "" {
			complete = false
		}
		if complete {
			completeNodes++
		}
		_ = requiredFields
	}

	if len(g.Nodes) > 0 {
		gs.MetadataCompleteness = float64(completeNodes) / float64(len(g.Nodes))
		gs.OwnerCoverage = float64(hasOwner) / float64(len(g.Nodes))
		gs.PermissionCompliance = float64(hasPermission) / float64(len(g.Nodes))
	} else {
		gs.MetadataCompleteness = 1.0
		gs.OwnerCoverage = 1.0
		gs.PermissionCompliance = 1.0
	}

	gs.Score = gs.MetadataCompleteness*0.5 + gs.OwnerCoverage*0.3 + gs.PermissionCompliance*0.2
	return gs
}

func assessReuseGrowth(g *graph.Graph, historyDir string) ReuseGrowthScore {
	rgs := ReuseGrowthScore{}

	prevAssessment := loadPreviousAssessment(historyDir)
	if prevAssessment != nil {
		growth := 0.0
		if prevAssessment.TotalScore > 0 {
			growth = (rgs.Score - prevAssessment.TotalScore) / prevAssessment.TotalScore
		}
		rgs.KnowledgeROI = math.Max(0, growth)
	}

	rgs.ReuseRate = 0.5
	rgs.ConversionRate = 0.3
	rgs.Score = rgs.ReuseRate*0.4 + rgs.ConversionRate*0.3 + math.Min(rgs.KnowledgeROI*10, 1.0)*0.3
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
