package configs

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Conf for config yaml
type Conf struct {
	Podcasts     map[string]Podcast `yaml:"podcasts"`
	CloudStorage struct {
		EndPointURL string `yaml:"endpoint_url"`
		Bucket      string `yaml:"bucket"`
		Region      string `yaml:"region"`
		Secrets     struct {
			Key    string `yaml:"aws_key"`
			Secret string `yaml:"aws_secret"`
		} `yaml:"secrets"`
	} `yaml:"cloud_storage"`
	Upload struct {
		ChunkSize int `yaml:"chunk_size"`
	} `yaml:"upload"`
	DB      string `yaml:"db"`
	Storage struct {
		Folder string `yaml:"folder"`
	} `yaml:"storage"`
}

// Podcast defines podcast section
type Podcast struct {
	Title   string `yaml:"title"`
	Folder  string `yaml:"folder"`
	MaxSize int64  `yaml:"max_size"`
}

// Load config from file
func Load(fileName string) (res *Conf, err error) {
	res = &Conf{}
	data, err := os.ReadFile(fileName) // nolint
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, res); err != nil {
		return nil, err
	}
	return res, nil
}
