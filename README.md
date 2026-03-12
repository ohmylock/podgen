# Podcast Generator

Podcast Generator is simple application for upload some episodes to s3 storage and generate feed to podcast player.

## Features

- Upload MP3 episodes to S3-compatible cloud storage
- Generate RSS/Atom feeds compatible with podcast players
- Automatic artwork generation - creates 3000x3000 gradient cover art when no podcast image exists
- MP3 metadata extraction - reads ID3 tags (title, artist, album, year, comment, duration) to enrich RSS feed descriptions
- Progress display - visual progress bar during uploads and deletions when running in terminal
- Rollback support - undo last upload or specific session
- Multiple podcast support from single configuration

## Usage

`podgen -s -u -p demopodcast, testpod`

```
Application options:
  -c, --conf=             config file (yml). Default podgen.yml
  -d, --db=               database file path (overrides config)
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

Help Options:
  -h, --help              Show this help message
```


## Configuration

Usually, podgen configuration is stored in `podgen.yml` file. It is a yaml file with the following structure:

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
| `PODGEN_CONF` | Path to config file (default: `podgen.yml`) |
| `PODGEN_DB` | Database file path (overrides config) |

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
- Uses a gradient background with colors derived from the podcast name
- Centers the podcast title with readable text
- Saves as `podcast.generated.png` in the podcast folder

Artwork generation is enabled by default. To disable:

```yaml
artwork:
  auto_generate: false
```

If disabled and no image exists, the `-i` (image upload) flag will fail with an error.