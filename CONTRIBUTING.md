# Contributing to Graphify Lens

Thanks for your interest in contributing!

## Development Setup

```bash
git clone https://github.com/cybernetix-lab/graphify-lens.git
cd graphify-lens
go mod tidy
```

## Project Structure

```
cmd/graphify-lens/     — Entry point, HTTP server
internal/
  api/            — REST API handlers
  config/         — Configuration management
  git/            — Git auto-commit manager
  graph/          — Knowledge graph parser
  quality/        — Quality assessment engine
  scheduler/      — Background task scheduler
web/
  static/         — Frontend (D3.js + Chart.js SPA)
  embed.go        — Go embed directive
data/sample/      — Sample knowledge base data
scripts/          — Build and distribution scripts
```

## Before Submitting

1. Run `go vet ./...` — no warnings
2. Run `go test ./...` — all tests pass
3. Run `go build ./...` — compiles cleanly
4. Format code with `gofmt -s -w .`

## Commit Convention

- `feat:` — new feature
- `fix:` — bug fix
- `docs:` — documentation
- `refactor:` — code restructuring
- `test:` — test additions/changes
- `chore:` — build, CI, dependencies

## Pull Request Process

1. Fork the repository
2. Create a feature branch
3. Make your changes with clear commit messages
4. Ensure all checks pass
5. Open a PR with a clear description of changes

## Code of Conduct

Be respectful. Be constructive. Assume good intent.
