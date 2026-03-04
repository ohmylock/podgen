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

	t.Run("missing storage folder", func(t *testing.T) {
		c := validConf()
		c.Storage.Folder = ""
		assert.ErrorContains(t, c.Validate(), "storage.folder")
	})

	t.Run("missing endpoint url", func(t *testing.T) {
		c := validConf()
		c.CloudStorage.EndPointURL = ""
		assert.ErrorContains(t, c.Validate(), "endpoint_url")
	})

	t.Run("missing bucket", func(t *testing.T) {
		c := validConf()
		c.CloudStorage.Bucket = ""
		assert.ErrorContains(t, c.Validate(), "bucket")
	})

	t.Run("no podcasts", func(t *testing.T) {
		c := validConf()
		c.Podcasts = nil
		assert.ErrorContains(t, c.Validate(), "no podcasts")
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
			name: "storage folder default",
			setup: func(c *Conf) {
				c.Storage.Folder = "/data"
			},
			expected: "/data/podgen.db",
		},
		{
			name: "absolute default",
			setup: func(c *Conf) {},
			expected: "podgen.db",
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
