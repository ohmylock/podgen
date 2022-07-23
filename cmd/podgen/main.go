package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jessevdk/go-flags"
	"podgen/internal/app/podgen"
	"podgen/internal/app/podgen/proc"
	"podgen/internal/configs"
)

var opts struct {
	Conf        string `short:"c" long:"conf" env:"PODGEN_CONF" default:"podgen.yml" description:"config file (yml)"`
	DB          string `short:"d" long:"db" env:"PODGEN_DB" description:"bolt db file"`
	Upload      bool   `short:"u" long:"upload" description:"Upload episodes"`
	Scan        bool   `short:"s" long:"scan" description:"Find and add new episodes"`
	UpdateFeed  bool   `short:"f" long:"feed" description:"Regenerate feeds"`
	UpdateImage bool   `short:"i" long:"image" description:"re upload cover of podcasts"`
	Podcasts    string `short:"p" long:"podcast" description:"Podcasts name (separator quota)"`
	AllPodcasts bool   `short:"a" long:"all" description:"All podcasts"`
	// Dbg bool `long:"dbg" env:"DEBUG" description:"show debug info"`
}

var version string

func main() {
	fmt.Printf("podgen %s\n", version)

	p := flags.NewParser(&opts, flags.PassDoubleDash|flags.HelpFlag)
	if _, err := p.Parse(); err != nil {
		if err.(*flags.Error).Type != flags.ErrHelp {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
		p.WriteHelp(os.Stderr)
		os.Exit(2)
	}

	configFile := opts.Conf
	if !proc.CheckFileExists(configFile) {
		configFile = "configs/podgen.yml"
	}

	if !proc.CheckFileExists(configFile) {
		log.Fatal("[ERROR] config file not found")
	}

	conf, err := configs.Load(configFile)
	if err != nil {
		log.Fatalf("[ERROR] can't load config %s, %v", opts.Conf, err)
	}

	if !proc.CheckFileExists(conf.Storage.Folder) {
		log.Fatal("[ERROR] storage folder not found")
	}

	dbFilepath := ""
	if conf.DB != "" {
		dbFilepath = conf.DB
	}

	if opts.DB != "" {
		dbFilepath = opts.DB
	}

	if dbFilepath == "" {
		log.Fatal("[ERROR] You don't set bolt db file")
	}

	db, err := podgen.NewBoltDB(dbFilepath)
	if err != nil {
		log.Fatalf("[ERROR] can't create boltdb instance, %v", err)
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
	procEntity := &proc.Processor{Storage: &proc.BoltDB{DB: db},
		S3Client:  &proc.S3Store{Client: s3client, Location: conf.CloudStorage.Region, Bucket: conf.CloudStorage.Bucket},
		Files:     &proc.Files{Storage: conf.Storage.Folder},
		ChunkSize: chunkSize,
	}

	app, err := podgen.NewApplication(conf, procEntity)
	if err != nil {
		log.Fatalf("[ERROR] can't create app, %v", err)
	}

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

	if opts.Scan {
		app.Update(podcasts)
	}

	var images map[string]string

	if opts.UpdateImage {
		images = app.UploadPodcastImage(podcasts)
	}

	if opts.Upload {
		app.DeleteOldEpisodes(podcasts)
		app.UploadEpisodes(podcasts)

		opts.UpdateFeed = true
	}

	if opts.UpdateFeed {
		if images == nil {
			images = app.GetPodcastImages(podcasts)
		}
		app.GenerateFeed(podcasts, images)
	}
}
