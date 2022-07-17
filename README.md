# Podcast Generator

Podcast Generator is simple application for upload some episodes to s3 storage and generate feed to podcast player.

## Usage

`podgen -s -u demopodcast, testpod`

```
Application options:
  -c, --conf= config file (yml). Default podgen.yml
  -s, --db= path to bolddb file. Defaut var/podgen.bdb
  -s, --scan= Find and add new episodes
  -u, --upload= Upload episodes by podcast name (separator quota)

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

upload:
  chunk_size: 3 # How meny episodes uploaded on stream
  
cloud_storage:
  endpoint_url: "storage.aws.com" # S3 storage endpoint url
  bucket: "YOU_SHOULD_SET_YOUR_S3_BUCKET_NAME" # S3 storage bucket
  region: "YOU_SHOULD_SET_YOUR_S3_REGION" # S3 storage region
  secrets:
    aws_key: "YOU_SHOULD_SET_YOUR_AWS_KEY" # S3 storage uploader aws key
    aws_secret: "YOU_SHOULD_SET_YOUR_AWS_SECRET" # S3 storage uploader aws secret