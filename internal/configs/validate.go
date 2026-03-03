package configs

import (
	"errors"
	"fmt"
)

// Validate checks the configuration for required fields.
func (c *Conf) Validate() error {
	if c.Storage.Folder == "" {
		return errors.New("storage.folder is required")
	}

	if c.CloudStorage.EndPointURL == "" {
		return errors.New("cloud_storage.endpoint_url is required")
	}

	if c.CloudStorage.Bucket == "" {
		return errors.New("cloud_storage.bucket is required")
	}

	if len(c.Podcasts) == 0 {
		return errors.New("no podcasts configured")
	}

	for id, p := range c.Podcasts {
		if p.Folder == "" {
			return fmt.Errorf("podcast %q: folder is required", id)
		}
	}

	return nil
}
