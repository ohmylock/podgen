# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.1] - 2026-03-12

### Added

- **XDG-compliant configuration:**
  - Default config location: `~/.config/podgen/config.yaml`
  - Default database location: `~/.config/podgen/podgen.db`
  - Auto-create config directory on first run
  - Config search priority: `--conf` flag → `~/.config/podgen/` → `./podgen.yml` → `./configs/podgen.yml`

## [0.1.0] - 2026-03-12

### Added

- **Podcast management:**
  - `--add-podcast` / `--add` to add podcasts from folders
  - `--title` flag for podcast title
  - `--list-podcasts` to show all podcasts
  - `--delete-podcast` to remove podcasts

- **Automatic artwork generation:**
  - Aurora mesh gradient style as default
  - Multiple styles: solid, gradient, radial, circles, blobs, noise, letter
  - `--artwork-style` flag for style selection
  - `-g` / `--generate-artwork` flag to force regenerate artwork

- **MP3 metadata extraction:**
  - ID3v2 tag reading (title, artist, album, year, comment)
  - Duration detection for `itunes:duration`
  - Date parsing from ID3 tags and filename patterns

- **Multiple storage backends:**
  - SQLite (default) with WAL mode
  - BoltDB (legacy support)
  - Migration between backends via `--migrate-from`

- **Configuration:**
  - `database` section for storage configuration
  - `artwork.auto_generate` option
  - Environment variable overrides (`PODGEN_CONF`, `PODGEN_DB`)

- **Other:**
  - Progress display during uploads/deletions
  - Graceful shutdown on SIGINT/SIGTERM

### Changed

- Default storage backend is now SQLite instead of BoltDB
- Updated Go to 1.23
- Improved RSS feed generation with richer metadata
- Migrated boltdb from unmaintained fork to go.etcd.io/bbolt

### Deprecated

- Legacy `db:` config field (use `database:` section instead)

## [0.0.5] - 2023-04-22

### Fixed

- Fixed get last episode for rollback status

## [0.0.4] - 2023-04-22

### Added

- Rollback episode and episodes statuses by session name
- Transaction support for database operations
- Comments to packages (lint compliance)

### Fixed

- Podcast's image path in feed
- Help description improvements

## [0.0.3] - 2022-11-30

### Added

- Podcast information display
- `--delete-old` flag to delete old podcasts
- goreleaser configuration

### Changed

- Refactored application flags

## [0.0.2] - 2022-07-23

### Added

- Upload podcast cover image
- Check if episode exists on S3 before uploading
- Default value for upload chunks

### Fixed

- README improvements

## [0.0.1] - 2022-07-19

### Added

- Initial release
- Upload MP3 episodes to S3-compatible storage
- Generate RSS/Atom feeds for podcast players
- Multiple podcast support from single configuration
- Queue upload episodes with chunk size configuration
- Delete old episodes functionality
- Storage folder configuration
- Basic test coverage

[0.1.1]: https://github.com/ohmylock/podgen/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/ohmylock/podgen/compare/v0.0.5...v0.1.0
[0.0.5]: https://github.com/ohmylock/podgen/compare/v0.0.4...v0.0.5
[0.0.4]: https://github.com/ohmylock/podgen/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/ohmylock/podgen/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/ohmylock/podgen/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/ohmylock/podgen/releases/tag/v0.0.1
