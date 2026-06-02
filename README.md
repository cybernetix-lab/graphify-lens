# Graphify Lens

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-macOS%20|%20Linux%20|%20Windows-lightgrey)]()

A self-hosted knowledge base management tool with built-in knowledge graph visualization, automated Git versioning, and quantitative quality assessment.

> **Graphify Lens turns your knowledge base into a measurable, version-controlled, and visually explorable asset.**

## Features

- **Knowledge Graph Visualization** — Interactive D3.js force-directed graph with zoom, drag, and node metadata inspection
- **Auto Git Versioning** — Background scheduler automatically commits knowledge base changes to Git
- **Quality Assessment** — Five-dimension scoring system (Coverage, Accuracy, Freshness, Governance, Reuse & Growth)
- **Quality Dashboard** — Real-time metrics dashboard with historical trend charts
- **Cross-Platform** — Single binary for macOS, Linux, and Windows (amd64/arm64)
- **Zero Dependencies** — No runtime dependencies, no database, no external services

## Quick Start

### Download

Download the latest binary from [Releases](https://github.com/cybernetix-lab/graphify-lens/releases) for your platform.

### Build from Source

```bash
git clone https://github.com/cybernetix-lab/graphify-lens.git
cd graphify-lens
go build -o graphify-lens ./cmd/graphify-lens/
```

### Run

```bash
# Start with default config
./graphify-lens

# Start with custom config
./graphify-lens --config config.example.json
```

Open **http://localhost:8080** in your browser.

### Add Knowledge Data

Place your knowledge nodes and edges in the work directory (default: `~/.graphify-lens/data/`):

```
~/.graphify-lens/data/
├── nodes/
│   ├── concept_001.json
│   ├── topic_001.json
│   └── ...
└── edges/
    └── relations.json
```

See [data/sample/](data/sample/) for example data.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    Browser (D3.js + Chart.js)        │
├─────────────────────────────────────────────────────┤
│                  HTTP API (net/http)                 │
├──────────┬──────────┬───────────┬───────────────────┤
│  Graph   │ Quality  │    Git    │    Scheduler      │
│  Parser  │ Assessor │  Manager  │  (time.Ticker)    │
├──────────┴──────────┴───────────┴───────────────────┤
│              File System (nodes/ + edges/)           │
└─────────────────────────────────────────────────────┘
```

## Quality Metrics

| Dimension | Weight | What It Measures |
|-----------|--------|------------------|
| Coverage | 30% | Type/topic/scenario coverage of the knowledge base |
| Accuracy | 25% | Edge confidence, source reference coverage |
| Freshness | 20% | Review compliance, stale page ratio |
| Governance | 15% | Metadata completeness, owner coverage, permission compliance |
| Reuse & Growth | 10% | Knowledge ROI, conversion rate |

## Configuration

```json
{
  "work_dir": "~/.graphify-lens/data",
  "port": 8080,
  "git_auto_commit": true,
  "commit_interval": "30m",
  "commit_message": "auto: graphify-lens knowledge base snapshot",
  "quality_history": "~/.graphify-lens/quality_history",
  "data_dir": "~/.graphify-lens",
  "author_name": "Graphify Lens Bot",
  "author_email": "graphify-lens-bot@teambuddy.local"
}
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/graph` | Full knowledge graph (nodes + edges) |
| GET | `/api/stats` | Graph statistics |
| GET | `/api/quality/current` | Current quality assessment |
| GET | `/api/quality/history` | Historical quality records |
| GET | `/api/status` | Scheduler status |
| POST | `/api/cycle/run` | Trigger assessment + commit cycle |

## Node Data Format

```json
{
  "id": "concept_001",
  "type": "concept",
  "title": "Knowledge Governance",
  "owner": "alice",
  "classification": "L1-team-public",
  "visibility_scope": "all_engineers",
  "status": "active",
  "last_reviewed_at": "2026-05-15T10:00:00Z",
  "freshness_sla": "720h",
  "source_refs": ["https://example.com/doc"],
  "scenario_tags": ["oncall", "coding"]
}
```

Supported node types: `concept`, `topic`, `runbook`, `decision`, `case`, `faq`, `article`, `source`.

## Cross-Platform Build

```bash
# Build for all platforms
bash scripts/build.sh

# Create distribution packages
bash scripts/dist.sh
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

## License

MIT — see [LICENSE](LICENSE) for details.
