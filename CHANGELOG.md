# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Official 0.2.0 release with cross-platform binaries
- Comprehensive documentation cleanup and simplification
- Streamlined usage guide focusing on core commands

### Changed
- Simplified installation instructions to point to GitHub releases
- Updated troubleshooting documentation to use `daemon` command instead of `ingest`
- Removed verbose architectural details from documentation
- Cleaned up introduction content for clarity

### Removed
- Contributing guidelines page (documentation simplification)
- In-documentation changelog (moved to GitHub releases only)
- References to research studies from introduction
- Verbose content aggregator statement from introduction

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

