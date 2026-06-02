package graph

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type NodeType string

const (
	NodeTypeConcept  NodeType = "concept"
	NodeTypeTopic    NodeType = "topic"
	NodeTypeRunbook  NodeType = "runbook"
	NodeTypeDecision NodeType = "decision"
	NodeTypeCase     NodeType = "case"
	NodeTypeFAQ      NodeType = "faq"
	NodeTypeArticle  NodeType = "article"
	NodeTypeSource   NodeType = "source"
)

type Node struct {
	ID             string            `json:"id"`
	Type           NodeType          `json:"type"`
	Title          string            `json:"title"`
	Owner          string            `json:"owner"`
	Classification string            `json:"classification"`
	Visibility     string            `json:"visibility_scope"`
	Status         string            `json:"status"`
	LastReviewedAt string            `json:"last_reviewed_at"`
	FreshnessSLA   string            `json:"freshness_sla"`
	SourceRefs     []string          `json:"source_refs"`
	ScenarioTags   []string          `json:"scenario_tags"`
	Properties     map[string]string `json:"properties"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

type Edge struct {
	ID         string `json:"id"`
	Source     string `json:"source"`
	Target     string `json:"target"`
	Relation   string `json:"relation"`
	Label      string `json:"label"`
	Confidence float64 `json:"confidence"`
}

type Graph struct {
	Nodes     []Node    `json:"nodes"`
	Edges     []Edge    `json:"edges"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   string    `json:"version"`
}

func ParseDir(workDir string) (*Graph, error) {
	g := &Graph{
		Nodes:     make([]Node, 0),
		Edges:     make([]Edge, 0),
		UpdatedAt: time.Now(),
		Version:   "1.0",
	}

	nodesDir := filepath.Join(workDir, "nodes")
	edgesDir := filepath.Join(workDir, "edges")

	if err := parseNodes(nodesDir, g); err != nil {
		return nil, err
	}
	if err := parseEdges(edgesDir, g); err != nil {
		return nil, err
	}

	return g, nil
}

func parseNodes(dir string, g *Graph) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var node Node
		if err := json.Unmarshal(data, &node); err != nil {
			continue
		}
		if node.ID == "" {
			node.ID = strings.TrimSuffix(entry.Name(), ".json")
		}
		g.Nodes = append(g.Nodes, node)
	}
	return nil
}

func parseEdges(dir string, g *Graph) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var edges []Edge
		if err := json.Unmarshal(data, &edges); err != nil {
			continue
		}
		g.Edges = append(g.Edges, edges...)
	}
	return nil
}

func (g *Graph) Stats() GraphStats {
	stats := GraphStats{
		TotalNodes:    len(g.Nodes),
		TotalEdges:    len(g.Edges),
		TypeBreakdown: make(map[NodeType]int),
		StatusBreakdown: map[string]int{
			"active":     0,
			"draft":      0,
			"deprecated": 0,
			"archived":   0,
			"unknown":    0,
		},
		ClassificationBreakdown: make(map[string]int),
		OwnerBreakdown:          make(map[string]int),
	}

	for _, n := range g.Nodes {
		stats.TypeBreakdown[n.Type]++
		status := n.Status
		if status == "" {
			status = "unknown"
		}
		stats.StatusBreakdown[status]++
		class := n.Classification
		if class == "" {
			class = "unset"
		}
		stats.ClassificationBreakdown[class]++
		owner := n.Owner
		if owner == "" {
			owner = "unset"
		}
		stats.OwnerBreakdown[owner]++
	}

	return stats
}

type GraphStats struct {
	TotalNodes              int              `json:"total_nodes"`
	TotalEdges              int              `json:"total_edges"`
	TypeBreakdown           map[NodeType]int `json:"type_breakdown"`
	StatusBreakdown         map[string]int   `json:"status_breakdown"`
	ClassificationBreakdown map[string]int   `json:"classification_breakdown"`
	OwnerBreakdown          map[string]int   `json:"owner_breakdown"`
}
