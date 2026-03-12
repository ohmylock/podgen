package configs

import (
	"errors"
	"fmt"
)

// Validate checks the configuration for required fields.
// Empty config is valid - podcasts can be added later via --add.
func (c *Conf) Validate() error {
	// Only validate cloud storage if podcasts exist
	if len(c.Podcasts) > 0 {
		if c.CloudStorage.EndPointURL == "" {
			return errors.New("cloud_storage.endpoint_url is required")
		}

		if c.CloudStorage.Bucket == "" {
			return errors.New("cloud_storage.bucket is required")
		}

		for id, p := range c.Podcasts {
			if p.Folder == "" {
				return fmt.Errorf("podcast %q: folder is required", id)
			}
		}
	}

	return nil
}

// ValidateForMigration checks only the configuration fields needed for migration.
// Migration only requires database settings, not podcast/S3 configuration.
func (c *Conf) ValidateForMigration() error {
	// Migration needs an explicitly configured destination database path.
	// We check raw config fields rather than GetStorageDSN() which has fallback defaults.
	// Without explicit config, migration would silently write to ./podgen.db
	if c.Database.DSN == "" && c.Database.Path == "" && c.DB == "" {
		return errors.New("database path is required for migration destination (set database.path, database.dsn, or legacy db field)")
	}
	return nil
}
