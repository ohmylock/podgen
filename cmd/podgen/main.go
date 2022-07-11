package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/jessevdk/go-flags"
	"podgen/internal/app/podgen"
	"podgen/internal/app/podgen/proc"
	"podgen/internal/configs"
)

var opts struct {
	Conf   string `short:"c" long:"conf" env:"PODGEN_CONF" default:"podgen.yml" description:"config file (yml)"`
	DB     string `short:"d" long:"db" env:"PODGEN_DB" default:"var/podgen.bdb" description:"bolt db file"`
	Upload bool   `short:"u" long:"upload" description:"Upload episodes"`
	Scan   bool   `short:"s" long:"scan" description:"Find and add new episodes"`

	// Dbg bool `long:"dbg" env:"DEBUG" description:"show debug info"`
}

func checkFileExists(filepath string) bool {
	if _, err := os.Stat(filepath); errors.Is(err, os.ErrNotExist) {
		return false
	}

	return true
}

func main() {
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

	if !checkFileExists(configFile) {
		configFile = "configs/podgen.yaml"

		if !checkFileExists(configFile) {
			log.Fatal("[ERROR] config file not found")
		}
	}

	conf, err := configs.Load(configFile)
	if err != nil {
		log.Fatalf("[ERROR] can't load config %s, %v", opts.Conf, err)
	}

	db, err := podgen.NewBoltDB(opts.DB)
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

	procEntity := &proc.Processor{Storage: &proc.BoltDB{DB: db},
		S3Client: &proc.S3Store{Client: s3client, Location: conf.CloudStorage.Region, Bucket: conf.CloudStorage.Bucket}}

	app, err := podgen.NewApplication(conf, procEntity)
	if err != nil {
		log.Fatalf("[ERROR] can't create app, %v", err)
	}

	if opts.Scan {
		app.Update()
	}

	if opts.Upload {
		app.Upload()
	}
}
