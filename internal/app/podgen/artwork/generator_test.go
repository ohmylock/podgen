package artwork

import (
	"crypto/sha256"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "cover.png")

	err := Generate("my-podcast", "My Podcast Title", out)
	require.NoError(t, err)

	info, err := os.Stat(out)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestGenerate_CorrectDimensions(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "cover.png")

	err := Generate("test-seed", "Test Podcast", out)
	require.NoError(t, err)

	f, err := os.Open(out) //nolint:gosec // test file path from t.TempDir()
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	cfg, _, err := image.DecodeConfig(f)
	require.NoError(t, err)
	assert.Equal(t, imageSize, cfg.Width)
	assert.Equal(t, imageSize, cfg.Height)
}

func TestGenerate_Deterministic(t *testing.T) {
	dir := t.TempDir()
	out1 := filepath.Join(dir, "cover1.png")
	out2 := filepath.Join(dir, "cover2.png")

	require.NoError(t, Generate("same-seed", "Same Title", out1))
	require.NoError(t, Generate("same-seed", "Same Title", out2))

	hash1 := fileHash(t, out1)
	hash2 := fileHash(t, out2)
	assert.Equal(t, hash1, hash2, "same seed and title must produce identical output")
}

func TestGenerate_DifferentSeeds(t *testing.T) {
	dir := t.TempDir()
	out1 := filepath.Join(dir, "cover1.png")
	out2 := filepath.Join(dir, "cover2.png")

	require.NoError(t, Generate("seed-alpha", "My Podcast", out1))
	require.NoError(t, Generate("seed-beta", "My Podcast", out2))

	hash1 := fileHash(t, out1)
	hash2 := fileHash(t, out2)
	assert.NotEqual(t, hash1, hash2, "different seeds must produce different output")
}

// fileHash returns the hex SHA-256 of a file's contents.
func fileHash(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path) //nolint:gosec // test file path from t.TempDir()
	require.NoError(t, err)
	return fmt.Sprintf("%x", sha256.Sum256(data))
}
