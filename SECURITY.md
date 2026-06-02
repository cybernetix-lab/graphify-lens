# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in Graphify Lens, please **do not** open a public issue.

Instead, email the maintainers directly. We will respond within 48 hours and work with you on a fix.

## Scope

Graphify Lens is a local-first knowledge base management tool. Security concerns typically involve:

- **Data exposure**: The tool serves knowledge graph data over HTTP. By default it binds to `localhost`. Do not expose it to untrusted networks without proper authentication.
- **Git operations**: The auto-commit feature executes `git` commands in the configured work directory. Ensure the work directory is trusted.
- **Configuration files**: The config file may contain paths and author information. Do not commit your personal `config.json` to public repositories.

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |
