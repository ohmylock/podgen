// Package configs for work with configurations
package configs

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultConfigDir returns the default configuration directory path.
// Returns ~/.config/podgen/ on all platforms.
func DefaultConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".config", "podgen")
	}
	return filepath.Join(home, ".config", "podgen")
}

// DefaultConfigFile returns the default config file path.
func DefaultConfigFile() string {
	return filepath.Join(DefaultConfigDir(), "config.yaml")
}

// DefaultDatabasePath returns the default database file path.
func DefaultDatabasePath() string {
	return filepath.Join(DefaultConfigDir(), "podgen.db")
}

// EnsureConfigDir creates the config directory if it doesn't exist.
func EnsureConfigDir() error {
	return os.MkdirAll(DefaultConfigDir(), 0o700)
}

// CreateDefaultConfig creates a template config file if it doesn't exist.
// Returns true if a new config was created, false if it already exists.
func CreateDefaultConfig() (bool, error) {
	configFile := DefaultConfigFile()

	// Check if config already exists
	if _, err := os.Stat(configFile); err == nil {
		return false, nil
	}

	// Ensure directory exists
	if err := EnsureConfigDir(); err != nil {
		return false, err
	}

	// Create template config
	template := `# podgen configuration
# Documentation: https://github.com/ohmylock/podgen

# Podcasts configuration
# Add podcasts using: podgen --add <folder> --title "Podcast Title"
podcasts: {}

# Local storage folder for MP3 files (default: current directory)
# storage:
#   folder: /path/to/podcasts

# S3-compatible cloud storage
cloud_storage:
  endpoint_url: ""
  bucket: ""
  region: ""
  secrets:
    aws_key: ""
    aws_secret: ""

# Database (SQLite by default, stored in ~/.config/podgen/podgen.db)
# database:
#   type: sqlite  # or: bolt
#   path: /custom/path/podgen.db

# Upload settings
# upload:
#   chunk_size: 3

# Artwork auto-generation (enabled by default)
# artwork:
#   auto_generate: true
`

	if err := os.WriteFile(configFile, []byte(template), 0o600); err != nil {
		return false, err
	}

	return true, nil
}

// StorageConfig defines database storage configuration
type StorageConfig struct {
	// Type specifies the storage backend: sqlite (default), bolt, or postgres
	Type string `yaml:"type"`
	// Path is the file path for sqlite/bolt databases
	Path string `yaml:"path"`
	// DSN is the connection string for postgres (overrides Path if set)
	DSN string `yaml:"dsn"`
}

// ArtworkConfig defines artwork auto-generation configuration
type ArtworkConfig struct {
	// AutoGenerate enables automatic podcast cover art generation when no artwork exists.
	// Nil/omitted defaults to true, explicit false disables it.
	AutoGenerate *bool `yaml:"auto_generate"`
}

// Conf for config yaml
type Conf struct {
	Podcasts     map[string]Podcast `yaml:"podcasts"`
	CloudStorage struct {
		EndPointURL string `yaml:"endpoint_url"`
		Bucket      string `yaml:"bucket"`
		Region      string `yaml:"region"`
		Secrets     struct {
			Key    string `yaml:"aws_key"`
			Secret string `yaml:"aws_secret"`
		} `yaml:"secrets"`
	} `yaml:"cloud_storage"`
	Upload struct {
		ChunkSize int `yaml:"chunk_size"`
	} `yaml:"upload"`
	DB      string `yaml:"db"` // Deprecated: use Database.Path instead
	Storage struct {
		Folder string `yaml:"folder"`
	} `yaml:"storage"`
	Database StorageConfig `yaml:"database"`
	Artwork  ArtworkConfig `yaml:"artwork"`
}

// Podcast defines podcast section
type Podcast struct {
	Title             string `yaml:"title"`
	Folder            string `yaml:"folder"`
	MaxSize           int64  `yaml:"max_size"`
	DeleteOldEpisodes bool   `yaml:"delete_old_episodes"`
	Info              struct {
		Author   string `yaml:"author"`
		Owner    string `yaml:"owner"`
		Email    string `yaml:"email"`
		Category string `yaml:"category"`
		Language string `yaml:"language"`
	} `yaml:"info"`
}

// Load config from file
func Load(fileName string) (res *Conf, err error) {
	res = &Conf{}
	data, err := os.ReadFile(fileName) // nolint
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, res); err != nil {
		return nil, err
	}
	return res, nil
}

// Save writes the configuration to a file
func (c *Conf) Save(fileName string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(fileName, data, 0o600)
}

// GetStorageType returns the configured storage type, defaulting to "sqlite"
// For backward compatibility, legacy DB field always defaults to "bolt" since
// the db: field historically only worked with Bolt backend
func (c *Conf) GetStorageType() string {
	if c.Database.Type != "" {
		return c.Database.Type
	}
	// Legacy backward-compatibility: db: field was Bolt-only, so default to bolt
	if c.DB != "" {
		return "bolt"
	}
	return "sqlite"
}

// HasStorageTypePreference returns true if config has explicit type preference,
// uses legacy db field, or explicitly sets database.path (indicating new config format).
// When any of these are set, CLI should not infer type from path extension.
func (c *Conf) HasStorageTypePreference() bool {
	return c.Database.Type != "" || c.Database.Path != "" || c.DB != ""
}

// InferStorageTypeFromPath detects storage type from file extension.
// Returns "bolt" for .bdb and .bolt files, "sqlite" otherwise.
func InferStorageTypeFromPath(path string) string {
	if strings.HasSuffix(path, ".bdb") || strings.HasSuffix(path, ".bolt") {
		return "bolt"
	}
	return "sqlite"
}

// GetStorageDSN returns the database path/DSN with fallback to legacy DB field
func (c *Conf) GetStorageDSN() string {
	// DSN takes priority (for postgres)
	if c.Database.DSN != "" {
		return c.Database.DSN
	}
	// Then Path from new config
	if c.Database.Path != "" {
		return c.Database.Path
	}
	// Fall back to legacy DB field
	if c.DB != "" {
		return c.DB
	}
	// Default: ~/.config/podgen/podgen.db
	return DefaultDatabasePath()
}

// IsArtworkAutoGenerateEnabled returns whether automatic artwork generation is enabled.
// Defaults to true if not explicitly configured.
func (c *Conf) IsArtworkAutoGenerateEnabled() bool {
	if c.Artwork.AutoGenerate == nil {
		return true
	}
	return *c.Artwork.AutoGenerate
}

// GetStorageFolder returns the storage folder path.
// Defaults to current directory if not configured.
func (c *Conf) GetStorageFolder() string {
	if c.Storage.Folder == "" {
		return "."
	}
	return c.Storage.Folder
}
