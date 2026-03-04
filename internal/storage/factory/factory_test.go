package factory_test

import (
	"os"
	"path/filepath"
	"testing"

	"podgen/internal/storage"
	"podgen/internal/storage/factory"
)

func TestNew_SQLite(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := storage.Config{
		Type:         storage.TypeSQLite,
		DSN:          dbPath,
		MaxOpenConns: 5,
		MaxIdleConns: 2,
	}

	store, err := factory.New(cfg)
	if err != nil {
		t.Fatalf("New(sqlite) failed: %v", err)
	}
	if store == nil {
		t.Fatal("New(sqlite) returned nil store")
	}

	// Verify we can open and close
	if err := store.Open(); err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer store.Close()

	// Verify SQLite WAL mode created the file
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("SQLite database file was not created")
	}
}

func TestNew_Bolt(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.bolt")

	cfg := storage.Config{
		Type: storage.TypeBolt,
		DSN:  dbPath,
	}

	store, err := factory.New(cfg)
	if err != nil {
		t.Fatalf("New(bolt) failed: %v", err)
	}
	if store == nil {
		t.Fatal("New(bolt) returned nil store")
	}

	// Verify we can open and close
	if err := store.Open(); err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer store.Close()

	// Verify BoltDB created the file
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("BoltDB database file was not created")
	}
}

func TestNew_Postgres(t *testing.T) {
	cfg := storage.Config{
		Type: storage.TypePostgres,
		DSN:  "postgres://localhost/test",
	}

	_, err := factory.New(cfg)
	if err == nil {
		t.Fatal("New(postgres) should return error for not implemented")
	}
	if err.Error() != "postgres backend: not implemented" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNew_UnknownType(t *testing.T) {
	cfg := storage.Config{
		Type: "unknown",
		DSN:  "/path/to/db",
	}

	_, err := factory.New(cfg)
	if err == nil {
		t.Fatal("New(unknown) should return error")
	}
}

func TestNewFromStrings(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		storageType string
		dsn         string
		wantErr     bool
	}{
		{
			name:        "sqlite explicit",
			storageType: "sqlite",
			dsn:         filepath.Join(tmpDir, "test1.db"),
			wantErr:     false,
		},
		{
			name:        "sqlite empty type defaults to sqlite",
			storageType: "",
			dsn:         filepath.Join(tmpDir, "test2.db"),
			wantErr:     false,
		},
		{
			name:        "bolt explicit",
			storageType: "bolt",
			dsn:         filepath.Join(tmpDir, "test3.bolt"),
			wantErr:     false,
		},
		{
			name:        "boltdb alias",
			storageType: "boltdb",
			dsn:         filepath.Join(tmpDir, "test4.bolt"),
			wantErr:     false,
		},
		{
			name:        "postgres not implemented",
			storageType: "postgres",
			dsn:         "postgres://localhost/test",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := factory.NewFromStrings(tt.storageType, tt.dsn)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if store == nil {
				t.Fatal("store is nil")
			}

			// Verify store can be opened
			if err := store.Open(); err != nil {
				t.Fatalf("Open() failed: %v", err)
			}
			store.Close()
		})
	}
}

func TestParseStorageType(t *testing.T) {
	tests := []struct {
		input string
		want  storage.StorageType
	}{
		{"sqlite", storage.TypeSQLite},
		{"SQLite", storage.TypeSQLite},
		{"SQLITE", storage.TypeSQLite},
		{"sqlite3", storage.TypeSQLite},
		{"", storage.TypeSQLite},
		{"  ", storage.TypeSQLite},
		{"bolt", storage.TypeBolt},
		{"boltdb", storage.TypeBolt},
		{"BoltDB", storage.TypeBolt},
		{"postgres", storage.TypePostgres},
		{"postgresql", storage.TypePostgres},
		{"pg", storage.TypePostgres},
		{"unknown", storage.TypeSQLite}, // defaults to sqlite
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := factory.ParseStorageType(tt.input)
			if got != tt.want {
				t.Errorf("ParseStorageType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFactoryCreatesWorkingStores(t *testing.T) {
	tmpDir := t.TempDir()

	backends := []struct {
		name string
		cfg  storage.Config
	}{
		{
			name: "sqlite",
			cfg: storage.Config{
				Type:         storage.TypeSQLite,
				DSN:          filepath.Join(tmpDir, "sqlite-test.db"),
				MaxOpenConns: 5,
				MaxIdleConns: 2,
			},
		},
		{
			name: "bolt",
			cfg: storage.Config{
				Type: storage.TypeBolt,
				DSN:  filepath.Join(tmpDir, "bolt-test.db"),
			},
		},
	}

	for _, bb := range backends {
		t.Run(bb.name, func(t *testing.T) {
			store, err := factory.New(bb.cfg)
			if err != nil {
				t.Fatalf("New() failed: %v", err)
			}

			if err := store.Open(); err != nil {
				t.Fatalf("Open() failed: %v", err)
			}
			defer store.Close()

			// Test basic operations work without error
			podcasts, err := store.ListPodcasts()
			if err != nil {
				t.Fatalf("ListPodcasts() failed: %v", err)
			}
			// Empty database returns nil or empty slice, both are valid
			if len(podcasts) != 0 {
				t.Errorf("ListPodcasts() returned %d podcasts, want 0", len(podcasts))
			}
		})
	}
}
