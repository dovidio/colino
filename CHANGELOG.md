# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.x] - Unreleased

## Added
- TUI interface for browsing your local content
- Content is now scraped and saved as markdown

## [0.2.0] - 2025-09-12

### Added
- Official 0.2.0 release with cross-platform binaries

### Fixed
- Database file is now properly created during initial setup
- Fixed references from old `ingest` command to new `daemon` command throughout the application

## [0.2.0-rc.1] - 2025-09-10

### Added
- Complete rewrite as Go binary
- Single binary providing both ingestion and MCP server
- Local SQLite database by default
- RSS ingestion with Trafilatura-based extraction
- YouTube transcript fetching
- MCP tools for LLM integration
- Cross-platform support

### Changed
- Replaced Python CLI with Go implementation
- Simplified configuration structure
- Consolidated daemon and MCP server functionality

### Breaking Changes
- Python CLI replaced, pip/pipx installs no longer supported
- Configuration keys simplified and paths standardized
- Summarization and filtering moved to LLM client

## [0.1.2] - 2025-15-09
Initial colino implementation written in python

