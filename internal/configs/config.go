// Package configs for work with configurations
package configs

import (
	"os"

	"gopkg.in/yaml.v3"
)

// StorageConfig defines database storage configuration
type StorageConfig struct {
	// Type specifies the storage backend: sqlite (default), bolt, or postgres
	Type string `yaml:"type"`
	// Path is the file path for sqlite/bolt databases
	Path string `yaml:"path"`
	// DSN is the connection string for postgres (overrides Path if set)
	DSN string `yaml:"dsn"`
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

// GetStorageType returns the configured storage type, defaulting to "sqlite"
func (c *Conf) GetStorageType() string {
	if c.Database.Type != "" {
		return c.Database.Type
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
	// Default: podgen.db in storage folder
	if c.Storage.Folder != "" {
		return c.Storage.Folder + "/podgen.db"
	}
	return "podgen.db"
}
