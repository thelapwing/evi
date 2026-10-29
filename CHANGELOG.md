# Changelog

All notable changes to EviDock are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Removed

- Optional line ranges in evidence declarations; each entry now covers a whole
  file and the lock records one SHA-256 hash per source file.

## [0.1.0] - 2026-10-22

### Added

- Initialize and tidy a workspace without overwriting existing files.
- Validate evidence declarations, referenced files, and optional line ranges.
- Create and verify SHA-256 integrity locks for registered evidence sources.
- Provide onboarding and project-guidance documents for AI coding agents.
- Run formatting checks, static analysis, race-enabled tests, and a build in
  GitHub Actions; publish text and HTML coverage reports as workflow artifacts.
