package podgen

import (
	"io/ioutil"
	"os"
	"testing"

	log "github.com/go-pkgz/lgr"
	"github.com/stretchr/testify/assert"
)

func TestNewBoltDB(t *testing.T) {
	tmpFile, _ := ioutil.TempFile("", "")
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			log.Fatalf("[ERROR] can't close temp file %s, %v", name, err)
		}
	}(tmpFile.Name())

	db, err := NewBoltDB(tmpFile.Name())
	assert.NoError(t, err)
	assert.NotNil(t, db)
}
