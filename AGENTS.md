# Repository Guidelines

## Project Structure & Module Organization
- Python package lives in `colino/` (entrypoint: `colino.main:main`). Key modules: `main.py`, `digest_manager.py`, `ingest_manager.py`, `db.py`, `config.py`, and sources under `colino/sources/{rss,youtube}.py`.
- Runtime config: `config.yaml` (repo root) or `~/.config/colino/config.yaml` (auto-created with defaults on first run).
- SQLite DB defaults to `~/Library/Application Support/Colino/colino.db` on macOS; `colino.db` otherwise.
- Docs website: `website/` (Docusaurus). Build artifacts: `dist/`.

## Build, Test, and Development Commands
- Setup: `poetry install` (requires Python 3.11+), optional `pre-commit install`.
- Lint: `poetry run ruff check .`  |  Format: `poetry run ruff format .`
- Types: `poetry run mypy colino/`
- Build package: `poetry build`
- Run CLI examples:
  - `poetry run colino ingest --all`
  - `poetry run colino list --hours 24`
  - `poetry run colino digest --rss` or `poetry run colino digest https://example.com`
- Website (docs): `cd website && npm ci && npm run start` | build: `npm run build`

## Coding Style & Naming Conventions
- Python only; 4-space indent; target line length 88 (Ruff). Auto-sort imports via Ruff (isort rules).
- Type hints required (mypy strict settings in `pyproject.toml`).
- Naming: `snake_case` for functions/vars, `PascalCase` for classes, `UPPER_CASE` for constants.
- Organize source adapters under `colino/sources/`; keep modules focused and small.

## Testing Guidelines
- No test suite yet. Prefer `pytest` with tests under `tests/` named `test_*.py`.
- Focus on unit tests for parsing, filtering, and DB operations. Use fixtures for sample feeds.
- When present, run: `pytest`; always ensure lint (`ruff`) and types (`mypy`) pass locally.

## Commit & Pull Request Guidelines
- Commits: concise, imperative subject; prefer Conventional Commits (e.g., `feat:`, `fix:`). Reference issues (`#123`) when applicable.
- PRs: include a clear description, linked issues, repro/validation steps, and screenshots for `website/` changes. Note config/env impacts (e.g., new keys).
- Requirements: pre-commit passes locally, CI green (Ruff, format check, mypy, package build).

## Security & Configuration Tips
- Set `OPENAI_API_KEY` via environment; never commit secrets or local DBs. Example: `export OPENAI_API_KEY=...`.
- macOS-only runtime; logs at `~/Library/Logs/Colino/colino.log`.
- Prefer user-level config in `~/.config/colino/config.yaml`; avoid committing `config.yaml` overrides.
