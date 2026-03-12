package podgen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"podgen/internal/app/podgen/proc"
	"podgen/internal/configs"
)

func TestNewApplication(t *testing.T) {
	conf := &configs.Conf{
		Podcasts: map[string]configs.Podcast{
			"test": {Title: "Test Podcast", Folder: "test"},
		},
	}

	procEntity := &proc.Processor{}

	app, err := NewApplication(conf, procEntity)
	require.NoError(t, err)
	assert.NotNil(t, app)
}

func TestApp_FindPodcasts(t *testing.T) {
	conf := &configs.Conf{
		Podcasts: map[string]configs.Podcast{
			"podcast1": {Title: "Podcast One", Folder: "p1"},
			"podcast2": {Title: "Podcast Two", Folder: "p2"},
		},
	}

	app, err := NewApplication(conf, &proc.Processor{})
	require.NoError(t, err)

	podcasts := app.FindPodcasts()
	assert.Len(t, podcasts, 2)
	assert.Contains(t, podcasts, "podcast1")
	assert.Contains(t, podcasts, "podcast2")
}
