<p align="right">
  <a href="README.ru.md">Читать на русском</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/go-1.25+-00ADD8?style=flat&logo=go" alt="Go version 1.25+">
  <a href="https://github.com/ohmylock/podgen/actions/workflows/ci.yml"><img src="https://github.com/ohmylock/podgen/actions/workflows/ci.yml/badge.svg" alt="CI Status"></a>
  <a href="https://github.com/ohmylock/podgen/releases"><img src="https://img.shields.io/github/v/release/ohmylock/podgen?include_prereleases" alt="Latest Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT License"></a>
  <a href="https://goreportcard.com/report/github.com/ohmylock/podgen"><img src="https://goreportcard.com/badge/github.com/ohmylock/podgen" alt="Go Report Card"></a>
</p>

<h1 align="center">podgen</h1>

<p align="center">
  <b>Podcast Generator — upload episodes to S3 and generate RSS feeds</b><br>
  <i>CLI tool for podcast management: MP3 scanning, metadata extraction, artwork generation</i>
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> •
  <a href="#features">Features</a> •
  <a href="#installation">Installation</a> •
  <a href="#usage">Usage</a> •
  <a href="#configuration">Configuration</a>
</p>

---

## What is podgen?

**podgen** is a command-line tool for podcast management. It uploads MP3 episodes to S3-compatible storage and generates RSS feeds compatible with Apple Podcasts, Spotify, and other podcast players.

## Quick Start

```bash
# Install
brew install ohmylock/tools/podgen

# Scan and upload episodes
podgen -s -u -p mypodcast

# All podcasts from config
podgen -s -u -a
```

## Features

- **S3 Upload** — any S3-compatible storage (AWS, Minio, Yandex Cloud, etc.)
- **RSS/Atom Feed** — compatible with Apple Podcasts, Spotify, Google Podcasts
- **Metadata Extraction** — ID3v2 tags: title, artist, album, year, duration
- **Artwork Generation** — 3000x3000 PNG with various gradient styles
- **Progress Bar** — visual upload progress in terminal
- **Rollback** — undo last upload or specific session
- **Multiple Podcasts** — single config for multiple podcasts
- **Graceful Shutdown** — clean exit on Ctrl+C

## Installation

### Homebrew (macOS/Linux)

```bash
brew install ohmylock/tools/podgen
```

### Download Binary

Download the appropriate binary for your platform from the [releases](https://github.com/ohmylock/podgen/releases) page:

| Platform | Architecture | File |
|----------|--------------|------|
| macOS | Apple Silicon | `podgen_*_darwin_arm64.tar.gz` |
| macOS | Intel | `podgen_*_darwin_amd64.tar.gz` |
| Linux | x86_64 | `podgen_*_linux_amd64.tar.gz` |
| Linux | ARM64 | `podgen_*_linux_arm64.tar.gz` |
| Windows | x86_64 | `podgen_*_windows_amd64.zip` |

### Go Install

```bash
go install github.com/ohmylock/podgen/cmd/podgen@latest
```

### Build from Source

```bash
git clone https://github.com/ohmylock/podgen.git
cd podgen
make build
# binary: bin/podgen
```

## Usage

`podgen -s -u -p demopodcast, testpod`

```
Application options:
  -c, --conf=             config file path (default: ~/.config/podgen/config.yaml)
  -d, --db=               database file path (default: ~/.config/podgen/podgen.db)
  -s, --scan              Find and add new episodes
  -u, --upload            Upload episodes
  -f, --feed              Regenerate feeds
  -i, --image             Upload podcast's cover
  -p, --podcast=          Podcasts name (separator quota)
  -a, --all               All podcasts
  -r, --rollback          Rollback last episode
      --rollback-session= Rollback by session name
      --rss               Show RSS feed URL for podcasts
      --migrate-from=     Migrate data from another database (format: type:path)
      --add-podcast=      Add new podcast from folder name
      --add=              Alias for --add-podcast
      --title=            Title for new podcast (used with --add-podcast)
      --clear             Force delete old episodes before upload
  -g, --generate-artwork  Force (re)generate podcast artwork
      --artwork-style=    Artwork style (solid, gradient, gradient-diagonal, radial, circles, blobs, noise, letter, aurora)

Help Options:
  -h, --help              Show this help message
```


## Configuration

### Config File Location

podgen searches for config file in the following order:

1. `--conf` flag or `PODGEN_CONF` environment variable
2. `~/.config/podgen/config.yaml` (recommended)
3. `./podgen.yml` (current directory, for backward compatibility)
4. `./configs/podgen.yml` (for backward compatibility)

The recommended location is `~/.config/podgen/config.yaml`. This keeps your configuration in a standard location and separates it from project files.

### Database Location

By default, the database is stored at `~/.config/podgen/podgen.db`. You can override this with:
- `database.path` in config file
- `-d` / `--db` flag
- `PODGEN_DB` environment variable

### Config File Format

The config file is YAML with the following structure:

```yaml
podcasts:
  demopodcast-example: # podcast name, can be repeated for multiple podcasts
    title: "Demo Podcast" # Podcast title
    folder: "demo" # Podcast where store episodes
    max_size: 10000000 # Optional. Max size limit to upload by once
    delete_old_episodes: true # Need to delete episodes before to upload new
    info: # Information in podcast feed
      author: user1 # Author of the podcast
      owner: user1 # Owner of the podcast
      email: podgen-user@localhost.com # Email of the owner of the podcast
      category: History # Podcast category. You can read all categories in apple support information https://podcasters.apple.com/support/1691-apple-podcasts-categories
      language: en # Optional. Language code for RSS feed (e.g., en, ru, de) 

database:
  type: "sqlite"        # Storage backend: sqlite (default) or bolt
  path: "podgen.db"     # File path for sqlite/bolt databases

# Legacy option (deprecated, use database section instead):
# db: "podgen.bdb"
# WARNING: The legacy db: field ALWAYS uses BoltDB regardless of file extension.
# To use SQLite, you MUST use the database section above.

storage:
  folder: "episodes" # Local folder where MP3 files are stored for scanning

upload:
  chunk_size: 3 # How many episodes uploaded on stream

artwork:
  auto_generate: true # Optional. Generate cover art if no image exists (default: true)

cloud_storage:
  endpoint_url: "s3.aws.com" # S3 storage endpoint url
  bucket: "podgen_bucket" # S3 storage bucket
  region: "central-eu1" # S3 storage region
  secrets:
    aws_key: "i8JFVo4fXxTCbqjU89" # S3 storage uploader aws key
    aws_secret: "egUiXQ6HFmmEY77r3j_W9ML74CkPHLw7P" # S3 storage uploader aws secret
```

## Environment Variables

Configuration can be overridden via environment variables:

| Variable | Description |
|----------|-------------|
| `PODGEN_CONF` | Path to config file (default: `~/.config/podgen/config.yaml`) |
| `PODGEN_DB` | Database file path (default: `~/.config/podgen/podgen.db`) |

Example:
```bash
PODGEN_CONF=/etc/podgen/config.yml PODGEN_DB=/var/lib/podgen/data.sqlite podgen -s -u -a
```

## Graceful Shutdown

Podgen handles `SIGINT` (Ctrl+C) and `SIGTERM` signals for graceful shutdown. When a signal is received during upload or other operations, the current operation completes before the application exits cleanly.

## Storage Backends

Podgen supports multiple database backends for storing episode metadata:

- **SQLite** (default) - Recommended for most users. Uses WAL mode for high performance and concurrent reads.
- **BoltDB** - Legacy embedded key-value store. Still supported for backwards compatibility.
- **PostgreSQL** (planned) - For production deployments requiring a dedicated database server. Not yet implemented.

### Configuring Storage

In `podgen.yml`:

```yaml
database:
  type: "sqlite"        # Options: sqlite, bolt
  path: "podgen.db"     # File path for sqlite/bolt
```

Or override via CLI: `podgen -d /path/to/database.db`

### Migrating Between Backends

To migrate data from one storage backend to another, use the `--migrate-from` flag with `-d` to specify the destination.

**Important:** The source and destination must be different files. BoltDB uses file locking, so you cannot migrate from a file that is also configured as the destination.

#### Migration from BoltDB to SQLite (Recommended)

If you have an existing BoltDB database and want to switch to SQLite:

```bash
# Step 1: Run migration (type is inferred from file extension)
podgen --migrate-from=bolt:/path/to/podgen.bdb -d /path/to/podgen.sqlite

# Step 2: Update your podgen.yml to use the new database
```

Update config from:
```yaml
db: "podgen.bdb"
```

To:
```yaml
database:
  type: "sqlite"
  path: "podgen.sqlite"
```

#### Other Migration Examples

```bash
# Migrate from SQLite to another SQLite database
podgen --migrate-from=sqlite:/path/to/source.db -d /path/to/dest.sqlite

# Migrate from BoltDB to another BoltDB file
podgen --migrate-from=bolt:/path/to/old.bdb -d /path/to/new.bdb
```

The database type is automatically inferred from the file extension:
- `.sqlite`, `.db` → SQLite
- `.bdb` → BoltDB

The migration copies all podcasts and episodes from the source to the destination database, preserving all metadata.

## MP3 Metadata Extraction

When scanning MP3 files, podgen automatically reads ID3v2 tags:

- Title - used as episode title in RSS feed (falls back to filename if empty)
- Artist, Album, Year - combined into episode description
- Comment - appended to description
- Duration - added as itunes:duration tag
- Year/Date - used for episode pubDate (falls back to date from filename pattern YYYY-MM-DD)

This allows your podcast feed to display rich metadata without manual configuration.

## Progress Display

When running in a terminal, podgen shows visual progress during uploads and deletions:

```
Uploading: episode-2024-01-15.mp3  [████████████░░░░░░░░]  60%  12.5 MB / 20.8 MB
```

Progress display is automatically disabled when output is piped or redirected.

## Automatic Artwork Generation

When no podcast cover image (`podcast.png`) exists in a podcast folder, podgen can automatically generate one. The generated artwork:

- Creates a 3000x3000 pixel PNG (meets Apple/Spotify requirements)
- Centers the podcast title with readable text
- Saves as `podcast.png` in the podcast folder

### Available Styles

| Style | Description |
|-------|-------------|
| `aurora` | Mesh gradient with vibrant colors, northern lights style (default). Each podcast gets unique colors based on its name |
| `letter` | Big first letter + soft gradient, muted pastel colors |
| `solid` | Solid pastel color |
| `gradient` | Vertical gradient, muted pastel |
| `gradient-diagonal` | Diagonal gradient, muted pastel |
| `radial` | Radial gradient (center lighter), muted pastel |
| `circles` | Soft translucent circles on pastel background |
| `blobs` | Organic blob shapes on pastel background |
| `noise` | Gradient with subtle texture, muted pastel |

### Usage

```bash
# Generate artwork when adding podcast (uses aurora by default)
podgen --add mypodcast

# Generate with specific style
podgen --add mypodcast --artwork-style=letter

# Force regenerate artwork (always overwrites existing)
podgen -g -p mypodcast

# Force regenerate with specific style
podgen -g -p mypodcast --artwork-style=blobs
```

### Configuration

Artwork generation is enabled by default. To disable auto-generation:

```yaml
artwork:
  auto_generate: false
```

If disabled and no image exists, the `-i` (image upload) flag will fail with an error.

Note: `-g` flag always regenerates artwork regardless of `auto_generate` setting.

## Adding Podcasts

To add a new podcast from an existing folder:

```bash
# With explicit title
podgen --add myfolder --title "My Podcast Title"

# Without title (folder name with capitalized first letter)
podgen --add myfolder
```

The folder must exist in the storage path. The podcast will be added to your config file.

If no `podcast.png` exists in the folder, artwork will be automatically generated.

## Makefile

```bash
make build          # compile bin/podgen
make test           # run tests with race detector
make cover          # tests with coverage report
make lint           # golangci-lint
make fmt            # format code
make install        # install to /usr/local/bin
make release        # goreleaser release
make release-check  # goreleaser dry-run
make clean          # remove build artifacts
```

## Contributing

Contributions are welcome! Please read the [contributing guide](CONTRIBUTING.md) before submitting a PR.

## License

MIT License — see [LICENSE](LICENSE) file.

---

<p align="center">
  Made with love for podcasters
</p>