# Podcast Generator

Podcast Generator is simple application for upload some episodes to s3 storage and generate feed to podcast player.

## Usage

`podgen -s -u demopodcast, testpod`

```
Application options:
  -c, --conf= config file (yml). Default podgen.yml
  -s, --db= path to bolddb file.
  -s, --scan= Find and add new episodes
  -u, --upload= Upload episodes by podcast name. Put codes of podcast from yaml file (separator quota)
  -i, --image= Upload podcast's cover. Put codes of podcast from yaml file (separator quota). You could put image to folder of podcast, and renamed to podcast.png

Help Options:
  -h, --help    Show this help message
```


## Configuration

Usually, podgen configuration is stored in `podgen.yml` file. It is a yaml file with the following structure:

```yaml
podcasts:
  demopodcast-example: # podcast name, can be repeated for multiple podcasts
    title: "Demo Podcast" # Podcast title
    folder: "demo" # Podcast where store episodes
    max_size: 10000000 # Optional. Max size limit to upload by once

db: "podgen.bdb" # Path to bolt db file
upload:
  chunk_size: 3 # How many episodes uploaded on stream
  
cloud_storage:
  endpoint_url: "s3.aws.com" # S3 storage endpoint url
  bucket: "podgen_bucket" # S3 storage bucket
  region: "central-eu1" # S3 storage region
  secrets:
    aws_key: "i8JFVo4fXxTCbqjU89" # S3 storage uploader aws key
    aws_secret: "egUiXQ6HFmmEY77r3j_W9ML74CkPHLw7P" # S3 storage uploader aws secret