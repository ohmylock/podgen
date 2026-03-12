# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Automatic artwork generation:**
  - Aurora mesh gradient style as default
  - Multiple styles: solid, gradient, radial, circles, blobs, noise, letter
  - `--artwork-style` flag for style selection
  - `-g` flag to force regenerate artwork

- **MP3 metadata extraction:**
  - ID3v2 tag reading (title, artist, album, year, comment)
  - Duration detection for `itunes:duration`
  - Date parsing from ID3 tags and filename patterns

- **Multiple storage backends:**
  - SQLite (default) with WAL mode
  - BoltDB (legacy support)
  - Migration between backends via `--migrate-from`

- **CLI enhancements:**
  - `--add-podcast` / `--add` to add podcasts from folders
  - `--title` flag for podcast title
  - `--generate-artwork` / `-g` flag
  - Progress display during uploads/deletions
  - Graceful shutdown on SIGINT/SIGTERM

- **Configuration:**
  - `database` section for storage configuration
  - `artwork.auto_generate` option
  - Environment variable overrides (`PODGEN_CONF`, `PODGEN_DB`)

### Changed

- Default storage backend is now SQLite instead of BoltDB
- Improved RSS feed generation with richer metadata

### Deprecated

- Legacy `db:` config field (use `database:` section instead)

## [0.1.0] - 2022-07-23

### Added

- Initial release
- Upload MP3 episodes to S3-compatible storage
- Generate RSS/Atom feeds for podcast players
- Multiple podcast support from single configuration
- Rollback support for undo operations

[Unreleased]: https://github.com/ohmylock/podgen/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/ohmylock/podgen/releases/tag/v0.1.0
