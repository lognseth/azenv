# Changelog

All notable changes to `azenv` will be documented in this file.

This project uses semantic versioning while the CLI behavior is still small and explicit.

## [0.2.0] - Unreleased

### Added

- Cobra-based subcommands and shell completion generation.
- Global `config.yaml` and per-context `metadata.yaml` files.
- Per-context Azure config directories under `contexts/<name>/azure`.
- Metadata refresh from Azure CLI account and tenant data.
- `login`, `logout`, `doctor`, `clone`, `rename`, and `starship` commands.
- `fzf` picker support for `azenv use` with no arguments.

### Changed

- `azenv exec <context> -- <command...>` runs commands inside a context without switching the current shell.

## [0.1.2] - 2026-07-09

### Changed

- Added a clearer hint when `azenv use <context>` is run without shell integration.
- Improved zsh/bash shell integration so `azenv use <context>` modifies the current shell.
- Made fish integration request fish-compatible shell output.

### Fixed

- `azenv current` now derives the active context from `AZURE_CONFIG_DIR`.
- `azenv ls` now marks the active context with `*` based on `AZURE_CONFIG_DIR`.
- Context path comparisons now tolerate equivalent cleaned paths.

## [0.1.1] - 2026-07-09

### Changed

- Bumped the displayed CLI version after the first round of context-selection fixes.

## [0.1.0] - 2026-07-09

### Added

- Initial Go CLI for managing isolated Azure CLI contexts with `AZURE_CONFIG_DIR`.
- Commands for `init`, `create`, `use`, `ls`, `current`, `rm`, `exec`, `path`, and `version`.
- Source installer script.
