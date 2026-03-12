// Package main start
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"unicode"

	"github.com/jessevdk/go-flags"
	"podgen/internal/app/podgen"
	"podgen/internal/app/podgen/artwork"
	"podgen/internal/app/podgen/proc"
	"podgen/internal/configs"
	"podgen/internal/pkg/progress"
	"podgen/internal/storage"
	"podgen/internal/storage/factory"
)

var opts struct {
	Conf              string `short:"c" long:"conf" env:"PODGEN_CONF" default:"podgen.yml" description:"config file (yml)"`
	DB                string `short:"d" long:"db" env:"PODGEN_DB" description:"database file path (overrides config)"`
	Upload            bool   `short:"u" long:"upload" description:"Upload episodes"`
	Scan              bool   `short:"s" long:"scan" description:"Find and add new episodes"`
	UpdateFeed        bool   `short:"f" long:"feed" description:"Regenerate feeds"`
	UpdateImage       bool   `short:"i" long:"image" description:"re upload cover of podcasts"`
	Podcasts          string `short:"p" long:"podcast" description:"Podcasts name (separator quota)"`
	AllPodcasts       bool   `short:"a" long:"all" description:"All podcasts"`
	Rollback          bool   `short:"r" long:"rollback" description:"Rollback last episode"`
	RollbackBySession string `long:"rollback-session" description:"Rollback by session name"`
	ShowRSS           bool   `long:"rss" description:"Show RSS feed URL for podcasts"`
	MigrateFrom       string `long:"migrate-from" description:"Migrate data from another database (format: type:path, e.g., bolt:/path/to/db)"`
	AddPodcast        string `long:"add-podcast" description:"Add new podcast from folder name"`
	AddPodcastAlias   string `long:"add" description:"Alias for --add-podcast"`
	PodcastTitle      string `long:"title" description:"Title for new podcast (used with --add-podcast)"`
	ForceDelete       bool   `long:"clear" description:"Force delete old episodes before upload (ignores delete_old_episodes setting)"`
	GenerateArtwork   bool   `short:"g" long:"generate-artwork" description:"Force (re)generate podcast artwork"`
	// Dbg bool `long:"dbg" env:"DEBUG" description:"show debug info"`
}

var version string

// isTerminal returns true if the given file is a terminal device.
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func main() {
	fmt.Printf("podgen %s\n", version)

	if !parseFlags() {
		return
	}

	// Merge --add alias into AddPodcast
	if opts.AddPodcastAlias != "" && opts.AddPodcast == "" {
		opts.AddPodcast = opts.AddPodcastAlias
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Handle migration if requested (uses simpler validation)
	if opts.MigrateFrom != "" {
		conf := loadConfig(true)
		if err := runMigration(conf); err != nil {
			log.Fatalf("[ERROR] migration failed: %v", err)
		}
		return
	}

	// Handle add-podcast command
	if opts.AddPodcast != "" {
		if err := runAddPodcast(); err != nil {
			log.Fatalf("[ERROR] %v", err)
		}
		return
	}

	conf := loadConfig(false)

	app, store := setupApplication(conf)
	defer func() { _ = store.Close() }()

	podcasts := resolvePodcasts(app)

	exitCode := runOperations(ctx, app, podcasts)
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func parseFlags() bool {
	p := flags.NewParser(&opts, flags.PassDoubleDash|flags.HelpFlag)
	_, err := p.Parse()
	if err != nil {
		if flagErr, ok := err.(*flags.Error); ok && flagErr.Type == flags.ErrHelp {
			p.WriteHelp(os.Stderr)
			return false
		}
		log.Fatalf("[ERROR] %v", err)
	}
	return true
}

func loadConfig(forMigration bool) *configs.Conf {
	configFile := opts.Conf
	if !proc.CheckFileExists(configFile) {
		configFile = "configs/podgen.yml"
	}

	configExists := proc.CheckFileExists(configFile)

	// Migration with -d flag can run without config file or with malformed config
	// The -d flag provides complete destination specification
	if forMigration && opts.DB != "" {
		if !configExists {
			return &configs.Conf{}
		}
		// Try to load config but don't fail if it's malformed
		conf, err := configs.Load(configFile)
		if err != nil {
			log.Printf("[WARN] config file could not be loaded: %v, using CLI flags only", err)
			return &configs.Conf{}
		}
		return conf
	}

	if !configExists {
		log.Fatal("[ERROR] config file not found")
	}

	conf, err := configs.Load(configFile)
	if err != nil {
		log.Fatalf("[ERROR] can't load config %s, %v", opts.Conf, err)
	}

	// Migration mode only needs database config, not full app config
	// If CLI -d flag is provided, it supplies the destination, so skip validation
	if forMigration {
		if opts.DB == "" {
			if err := conf.ValidateForMigration(); err != nil {
				log.Fatalf("[ERROR] invalid config for migration: %v", err)
			}
		}
		return conf
	}

	if err := conf.Validate(); err != nil {
		log.Fatalf("[ERROR] invalid config: %v", err)
	}

	if !proc.CheckFileExists(conf.Storage.Folder) {
		log.Fatal("[ERROR] storage folder not found")
	}

	return conf
}

func setupApplication(conf *configs.Conf) (*podgen.App, storage.Store) {
	// Create storage using factory
	storageType := conf.GetStorageType()
	storageDSN := conf.GetStorageDSN()

	// CLI flag overrides config path only, not type
	// Type is determined by config (explicit or legacy inference), CLI only changes path
	if opts.DB != "" {
		storageDSN = opts.DB
	}

	if storageDSN == "" {
		log.Fatal("[ERROR] database path not configured")
	}

	store, err := factory.NewFromStrings(storageType, storageDSN)
	if err != nil {
		log.Fatalf("[ERROR] can't create storage instance: %v", err)
	}

	if err := store.Open(); err != nil {
		log.Fatalf("[ERROR] can't open storage: %v", err)
	}

	s3client, err := podgen.NewS3Client(
		conf.CloudStorage.EndPointURL,
		conf.CloudStorage.Secrets.Key,
		conf.CloudStorage.Secrets.Secret,
		true)
	if err != nil {
		log.Fatalf("[ERROR] can't create s3client instance, %v", err)
	}

	chunkSize := conf.Upload.ChunkSize
	if chunkSize == 0 {
		chunkSize = 3
	}

	procEntity := &proc.Processor{
		Storage:     store,
		S3Client:    &proc.S3Store{Client: s3client, Location: conf.CloudStorage.Region, Bucket: conf.CloudStorage.Bucket},
		Files:       &proc.Files{Storage: conf.Storage.Folder},
		StoragePath: conf.Storage.Folder,
		ChunkSize:   chunkSize,
	}

	if isTerminal(os.Stdout) {
		procEntity.Progress = progress.NewMulti(os.Stdout, chunkSize, 0)
	}

	app, err := podgen.NewApplication(conf, procEntity)
	if err != nil {
		log.Fatalf("[ERROR] can't create app, %v", err)
	}

	return app, store
}

// runMigration migrates data from a source database to the configured destination.
// The source format is "type:path" (e.g., "bolt:/path/to/db" or "sqlite:/path/to/db.sqlite").
func runMigration(conf *configs.Conf) error {
	// Parse source specification
	srcSpec := opts.MigrateFrom
	var srcType, srcPath string

	// Find the first colon that separates type from path
	for i, c := range srcSpec {
		if c == ':' {
			srcType = srcSpec[:i]
			srcPath = srcSpec[i+1:]
			break
		}
	}

	if srcType == "" || srcPath == "" {
		return fmt.Errorf("invalid migration source format: %q (expected type:path, e.g., bolt:/path/to/db)", srcSpec)
	}

	// Create source store
	srcStore, err := factory.NewFromStrings(srcType, srcPath)
	if err != nil {
		return fmt.Errorf("failed to create source store: %w", err)
	}
	if err := srcStore.Open(); err != nil {
		return fmt.Errorf("failed to open source store: %w", err)
	}
	defer func() { _ = srcStore.Close() }()

	// Create destination store from config
	// When -d flag is provided for migration, always infer type from CLI path
	// This allows migrating from Bolt (legacy config) to SQLite (new default)
	dstType := conf.GetStorageType()
	dstPath := conf.GetStorageDSN()
	if opts.DB != "" {
		dstPath = opts.DB
		// For migration with explicit -d, always infer type from destination path
		// CLI flag takes precedence over config
		dstType = configs.InferStorageTypeFromPath(opts.DB)
	}

	dstStore, err := factory.NewFromStrings(dstType, dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination store: %w", err)
	}
	if err := dstStore.Open(); err != nil {
		return fmt.Errorf("failed to open destination store: %w", err)
	}
	defer func() { _ = dstStore.Close() }()

	log.Printf("[INFO] Migrating from %s (%s) to %s (%s)", srcType, srcPath, dstType, dstPath)

	// Run migration
	stats, err := storage.Migrate(srcStore, dstStore)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	log.Printf("[INFO] Migration complete: %d podcasts (%d failed), %d episodes migrated, %d failed",
		stats.PodcastsProcessed, stats.PodcastsFailed, stats.EpisodesMigrated, stats.EpisodesFailed)

	return nil
}

// runAddPodcast adds a new podcast entry to the config file.
func runAddPodcast() error {
	folderName := opts.AddPodcast

	// Validate folder name to prevent path traversal attacks
	// Must be a simple directory name: no separators, not absolute, not "." or ".."
	if folderName == "" || folderName == "." || folderName == ".." {
		return fmt.Errorf("invalid folder name %q: cannot be empty or special directory", folderName)
	}
	if filepath.IsAbs(folderName) || strings.ContainsAny(folderName, `/\`) {
		return fmt.Errorf("invalid folder name %q: must be a simple directory name without path separators", folderName)
	}
	// After cleaning, verify it's still a simple name (no internal traversal)
	cleanName := filepath.Clean(folderName)
	if cleanName != folderName || cleanName == "." || cleanName == ".." {
		return fmt.Errorf("invalid folder name %q: contains invalid path components", folderName)
	}

	podcastID := strings.ToLower(folderName)

	// Load existing config
	configFile := opts.Conf
	if !proc.CheckFileExists(configFile) {
		configFile = "configs/podgen.yml"
	}
	if !proc.CheckFileExists(configFile) {
		return fmt.Errorf("config file not found: %s", opts.Conf)
	}

	conf, err := configs.Load(configFile)
	if err != nil {
		return fmt.Errorf("can't load config: %w", err)
	}

	// Check if podcast ID already exists
	if _, exists := conf.Podcasts[podcastID]; exists {
		return fmt.Errorf("podcast %q already exists in config", podcastID)
	}

	// Check if folder exists in storage
	storagePath := conf.Storage.Folder
	if storagePath == "" {
		storagePath = "storage"
	}
	folderPath := filepath.Join(storagePath, folderName)
	if !proc.CheckFileExists(folderPath) {
		return fmt.Errorf("folder %q not found in storage (%s)", folderName, storagePath)
	}

	// Generate title: capitalize first letter if not provided
	title := opts.PodcastTitle
	if title == "" {
		runes := []rune(folderName)
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
		}
		title = string(runes)
	}

	// Generate artwork if no podcast image exists
	podcastImagePath := filepath.Join(folderPath, "podcast.png")
	if !proc.CheckFileExists(podcastImagePath) {
		log.Printf("[INFO] Generating artwork for %s", podcastID)
		if err := artwork.Generate(podcastID, title, podcastImagePath); err != nil {
			return fmt.Errorf("can't generate artwork: %w", err)
		}
		log.Printf("[INFO] Artwork generated at %s", podcastImagePath)
	}

	// Initialize podcasts map if nil
	if conf.Podcasts == nil {
		conf.Podcasts = make(map[string]configs.Podcast)
	}

	// Create new podcast entry
	conf.Podcasts[podcastID] = configs.Podcast{
		Title:  title,
		Folder: folderName,
	}

	// Save config
	if err := conf.Save(configFile); err != nil {
		return fmt.Errorf("can't save config: %w", err)
	}

	log.Printf("[INFO] Added podcast %q (title: %q, folder: %q) to %s", podcastID, title, folderName, configFile)
	return nil
}

func resolvePodcasts(app *podgen.App) string {
	podcasts := opts.Podcasts
	if podcasts == "" && opts.AllPodcasts {
		podcastEntities := app.FindPodcasts()
		for i := range podcastEntities {
			if podcasts != "" {
				podcasts += ", "
			}
			podcasts += i
		}
	}

	if podcasts == "" {
		log.Fatalf("[ERROR] You didn't list podcasts")
	}
	return podcasts
}

func runOperations(ctx context.Context, app *podgen.App, podcasts string) int {
	var hasError bool

	if opts.Scan {
		if err := app.Update(ctx, podcasts); err != nil {
			hasError = true
		}
	}

	var images map[string]string
	if opts.UpdateImage || opts.GenerateArtwork {
		images = app.UploadPodcastImage(ctx, podcasts, opts.GenerateArtwork)
	}

	if opts.Rollback {
		app.RollbackEpisodes(ctx, podcasts)
	} else if opts.RollbackBySession != "" {
		app.RollbackEpisodesBySession(ctx, podcasts, opts.RollbackBySession)
	}

	if opts.Upload {
		// Auto-scan before upload to find new episodes
		if err := app.Update(ctx, podcasts); err != nil {
			hasError = true
		}
		if err := app.DeleteOldEpisodes(ctx, podcasts, opts.ForceDelete); err != nil {
			hasError = true
		}
		if err := app.UploadEpisodes(ctx, podcasts); err != nil {
			hasError = true
		}
		// Always auto-trigger feed update after upload phase, even if some podcasts failed
		// Each podcast's feed will be regenerated from its currently uploaded episodes
		opts.UpdateFeed = true
	}

	// Run feed generation if explicitly requested OR auto-triggered by upload
	if opts.UpdateFeed {
		if images == nil {
			images = app.GetPodcastImages(ctx, podcasts)
		}
		if err := app.GenerateFeed(ctx, podcasts, images); err != nil {
			hasError = true
		}
	}

	if opts.ShowRSS {
		urls := app.GetFeedURLs(podcasts)
		for id, url := range urls {
			fmt.Printf("%s: %s\n", id, url)
		}
	}

	if hasError {
		return 1
	}
	return 0
}
