package graph

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type GraphifyNode struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	FileType       string `json:"file_type"`
	SourceFile     string `json:"source_file"`
	SourceLocation string `json:"source_location,omitempty"`
	SourceURL      string `json:"source_url,omitempty"`
	CapturedAt     string `json:"captured_at,omitempty"`
	Author         string `json:"author,omitempty"`
	Contributor    string `json:"contributor,omitempty"`
}

type GraphifyEdge struct {
	Source          string  `json:"source"`
	Target          string  `json:"target"`
	Relation        string  `json:"relation"`
	Confidence      string  `json:"confidence"`
	ConfidenceScore float64 `json:"confidence_score"`
	SourceFile      string  `json:"source_file,omitempty"`
	SourceLocation  string  `json:"source_location,omitempty"`
	Weight          float64 `json:"weight"`
}

type GraphifyData struct {
	Directed   bool           `json:"directed"`
	Multigraph bool           `json:"multigraph"`
	Graph      map[string]any `json:"graph"`
	Nodes      []GraphifyNode `json:"nodes"`
	Links      []GraphifyEdge `json:"links"`
}

type Graph struct {
	Nodes     []GraphifyNode `json:"nodes"`
	Edges     []GraphifyEdge `json:"edges"`
	UpdatedAt time.Time      `json:"updated_at"`
	Version   string         `json:"version"`
}

func ParseGraphifyOut(workDir string) (*Graph, error) {
	graphPath := filepath.Join(workDir, "graphify-out", "graph.json")
	data, err := os.ReadFile(graphPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Graph{
				Nodes:     make([]GraphifyNode, 0),
				Edges:     make([]GraphifyEdge, 0),
				UpdatedAt: time.Now(),
				Version:   "1.0",
			}, nil
		}
		return nil, err
	}

	var raw GraphifyData
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	g := &Graph{
		Nodes:     raw.Nodes,
		Edges:     raw.Links,
		UpdatedAt: time.Now(),
		Version:   "1.0",
	}

	if g.Nodes == nil {
		g.Nodes = make([]GraphifyNode, 0)
	}
	if g.Edges == nil {
		g.Edges = make([]GraphifyEdge, 0)
	}

	return g, nil
}

func (g *Graph) Stats() GraphStats {
	stats := GraphStats{
		TotalNodes:          len(g.Nodes),
		TotalEdges:          len(g.Edges),
		FileTypeBreakdown:   make(map[string]int),
		ConfidenceBreakdown: make(map[string]int),
		AuthorBreakdown:     make(map[string]int),
	}

	for _, n := range g.Nodes {
		ft := n.FileType
		if ft == "" {
			ft = "unknown"
		}
		stats.FileTypeBreakdown[ft]++

		author := n.Author
		if author == "" {
			author = "unset"
		}
		stats.AuthorBreakdown[author]++
	}

	for _, e := range g.Edges {
		conf := e.Confidence
		if conf == "" {
			conf = "UNKNOWN"
		}
		stats.ConfidenceBreakdown[conf]++
	}

	return stats
}

type GraphStats struct {
	TotalNodes          int            `json:"total_nodes"`
	TotalEdges          int            `json:"total_edges"`
	FileTypeBreakdown   map[string]int `json:"file_type_breakdown"`
	ConfidenceBreakdown map[string]int `json:"confidence_breakdown"`
	AuthorBreakdown     map[string]int `json:"author_breakdown"`
}
