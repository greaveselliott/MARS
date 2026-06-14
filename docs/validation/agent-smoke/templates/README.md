# Agent Smoke Project Templates

Template directories describe the expected generated shapes for ephemeral
targets. They are documentation and recipe anchors only; the smoke runner does
not store generated repos here.

Project types:

- `static-web`: raw `index.html`, `style.css`, and `app.js`.
- `react-web`: package-managed React/Vite app.
- `browser-game-phaser`: package-managed Phaser browser game.
- `canvas-game-vanilla`: no-framework canvas game.
- `go-api`: standard-library Go HTTP API.
- `go-cli`: Go command-line tool.
- `go-library`: Go library package.
- `docs-site`: documentation/content workflow target.
- `existing-maintenance`: generated project plus stale tickets, failed checks,
  old reports, or documentation drift.
