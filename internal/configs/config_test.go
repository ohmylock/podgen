package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
