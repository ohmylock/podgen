package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validConf() *Conf {
	c := &Conf{}
	c.Storage.Folder = "/data"
	c.CloudStorage.EndPointURL = "https://s3.example.com"
	c.CloudStorage.Bucket = "my-bucket"
	c.Podcasts = map[string]Podcast{
		"p1": {Title: "Test", Folder: "folder1"},
	}
	return c
}

func TestConf_Validate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		require.NoError(t, validConf().Validate())
	})

	t.Run("empty config is valid", func(t *testing.T) {
		c := &Conf{}
		require.NoError(t, c.Validate())
	})

	t.Run("missing endpoint url with podcasts", func(t *testing.T) {
		c := validConf()
		c.CloudStorage.EndPointURL = ""
		assert.ErrorContains(t, c.Validate(), "endpoint_url")
	})

	t.Run("missing bucket with podcasts", func(t *testing.T) {
		c := validConf()
		c.CloudStorage.Bucket = ""
		assert.ErrorContains(t, c.Validate(), "bucket")
	})

	t.Run("no podcasts is valid", func(t *testing.T) {
		c := validConf()
		c.Podcasts = nil
		require.NoError(t, c.Validate())
	})

	t.Run("podcast missing folder", func(t *testing.T) {
		c := validConf()
		c.Podcasts["p1"] = Podcast{Title: "Test", Folder: ""}
		assert.ErrorContains(t, c.Validate(), "folder is required")
	})
}

func TestLoad(t *testing.T) {
	conf, err := Load("testdata/config.yml")
	require.NoError(t, err)

	assert.Equal(t, len(conf.Podcasts), 2)

	assert.Equal(t, conf.CloudStorage.EndPointURL, "storage_url")
	assert.Equal(t, conf.CloudStorage.Bucket, "bucket_name")
	assert.Equal(t, conf.CloudStorage.Region, "region-us")
	assert.Equal(t, conf.CloudStorage.Secrets.Key, "123123123")
	assert.Equal(t, conf.CloudStorage.Secrets.Secret, "abc123123123xyz")
}

func TestLoadConfigNotFound(t *testing.T) {
	conf, err := Load("/tmp/test-bestow-nautch-toss-fritter-pygmy-unrest.yml")
	assert.Nil(t, conf)
	assert.EqualError(t, err, "open /tmp/test-bestow-nautch-toss-fritter-pygmy-unrest.yml: no such file or directory")
}

func TestGetStorageType(t *testing.T) {
	tests := []struct {
		name     string
		dbType   string
		expected string
	}{
		{"empty type defaults to sqlite", "", "sqlite"},
		{"explicit sqlite", "sqlite", "sqlite"},
		{"explicit bolt", "bolt", "bolt"},
		{"explicit postgres", "postgres", "postgres"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Conf{}
			c.Database.Type = tt.dbType
			assert.Equal(t, tt.expected, c.GetStorageType())
		})
	}

	// Test legacy DB field with bolt extension
	t.Run("legacy DB with .bdb extension defaults to bolt", func(t *testing.T) {
		c := &Conf{}
		c.DB = "/path/to/legacy.bdb"
		assert.Equal(t, "bolt", c.GetStorageType())
	})

	t.Run("legacy DB with .bolt extension defaults to bolt", func(t *testing.T) {
		c := &Conf{}
		c.DB = "/path/to/legacy.bolt"
		assert.Equal(t, "bolt", c.GetStorageType())
	})

	t.Run("legacy DB field always defaults to bolt (backward compat)", func(t *testing.T) {
		c := &Conf{}
		c.DB = "/path/to/legacy.db"
		// Legacy db: field was Bolt-only, so always default to bolt regardless of extension
		assert.Equal(t, "bolt", c.GetStorageType())
	})

	t.Run("explicit type takes priority over legacy DB extension", func(t *testing.T) {
		c := &Conf{}
		c.Database.Type = "sqlite"
		c.DB = "/path/to/legacy.bdb"
		assert.Equal(t, "sqlite", c.GetStorageType())
	})
}

func TestInferStorageTypeFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"bdb extension", "/path/to/data.bdb", "bolt"},
		{"bolt extension", "/path/to/data.bolt", "bolt"},
		{"db extension defaults to sqlite", "/path/to/data.db", "sqlite"},
		{"sqlite extension defaults to sqlite", "/path/to/data.sqlite", "sqlite"},
		{"no extension defaults to sqlite", "/path/to/data", "sqlite"},
		{"empty path", "", "sqlite"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, InferStorageTypeFromPath(tt.path))
		})
	}
}

func TestHasStorageTypePreference(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Conf)
		expected bool
	}{
		{
			name:     "empty config has no preference",
			setup:    func(c *Conf) {},
			expected: false,
		},
		{
			name: "explicit type has preference",
			setup: func(c *Conf) {
				c.Database.Type = "sqlite"
			},
			expected: true,
		},
		{
			name: "legacy DB field has preference",
			setup: func(c *Conf) {
				c.DB = "/path/to/legacy.bdb"
			},
			expected: true,
		},
		{
			name: "database.path has preference (new config format)",
			setup: func(c *Conf) {
				c.Database.Path = "/path/to/podgen.db"
			},
			expected: true,
		},
		{
			name: "both explicit and legacy has preference",
			setup: func(c *Conf) {
				c.Database.Type = "bolt"
				c.DB = "/path/to/legacy.bdb"
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Conf{}
			tt.setup(c)
			assert.Equal(t, tt.expected, c.HasStorageTypePreference())
		})
	}
}

func TestIsArtworkAutoGenerateEnabled(t *testing.T) {
	t.Run("nil defaults to true", func(t *testing.T) {
		c := &Conf{}
		assert.True(t, c.IsArtworkAutoGenerateEnabled())
	})

	t.Run("explicit true", func(t *testing.T) {
		c := &Conf{}
		v := true
		c.Artwork.AutoGenerate = &v
		assert.True(t, c.IsArtworkAutoGenerateEnabled())
	})

	t.Run("explicit false", func(t *testing.T) {
		c := &Conf{}
		v := false
		c.Artwork.AutoGenerate = &v
		assert.False(t, c.IsArtworkAutoGenerateEnabled())
	})
}

func TestGetStorageDSN(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(*Conf)
		expected      string
	}{
		{
			name: "DSN takes priority",
			setup: func(c *Conf) {
				c.Database.DSN = "postgres://localhost/db"
				c.Database.Path = "/path/to/db"
				c.DB = "/legacy/db"
			},
			expected: "postgres://localhost/db",
		},
		{
			name: "Path when no DSN",
			setup: func(c *Conf) {
				c.Database.Path = "/path/to/podgen.db"
				c.DB = "/legacy/db"
			},
			expected: "/path/to/podgen.db",
		},
		{
			name: "legacy DB fallback",
			setup: func(c *Conf) {
				c.DB = "/legacy/podgen.bolt"
			},
			expected: "/legacy/podgen.bolt",
		},
		{
			name: "default to config dir",
			setup: func(c *Conf) {},
			expected: DefaultDatabasePath(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Conf{}
			tt.setup(c)
			assert.Equal(t, tt.expected, c.GetStorageDSN())
		})
	}
}
