# Podcast Generator

Podcast Generator is simple application for upload some episodes to s3 storage and generate feed to podcast player.

## Features

- Upload MP3 episodes to S3-compatible cloud storage
- Generate RSS/Atom feeds compatible with podcast players
- MP3 metadata extraction - reads ID3 tags (title, artist, album, year, comment, duration) to enrich RSS feed descriptions
- Progress display - visual progress bar during uploads and deletions when running in terminal
- Rollback support - undo last upload or specific session
- Multiple podcast support from single configuration

## Usage

`podgen -s -u -p demopodcast, testpod`

```
Application options:
  -c, --conf= config file (yml). Default podgen.yml
  -d, --db= path to bolddb file.
  -s, --scan= Find and add new episodes.
  -u, --upload= Upload episodes.
  -i, --image= Upload podcast's cover.
  -p, --podcast= Put podcasts code from yaml file (separator quota)
  -a, --all All podcasts.
  -r, --rollback          Rollback last episode
      --rollback-session= Rollback by session name
  

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
    delete_old_episodes: true # Need to delete episodes before to upload new
    info: # Information in podcast feed
      author: user1 # Author of the podcast 
      owner: user1 # Owner of the podcast
      email: podgen-user@localhost.com # Email of the owner of the podcast
      category: History # Podcast category. You can read all categories in apple support information https://podcasters.apple.com/support/1691-apple-podcasts-categories 

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
```

## MP3 Metadata Extraction

When scanning MP3 files, podgen automatically reads ID3v2 tags:

- Title - used as episode title in RSS feed (falls back to filename if empty)
- Artist, Album, Year - combined into episode description
- Comment - appended to description
- Duration - added as itunes:duration tag
- Year/Date - used for episode pubDate (falls back to date from filename pattern YYYY-MM-DD)

This allows your podcast feed to display rich metadata without manual configuration.

## Progress Display

When running in a terminal, podgen shows visual progress during uploads and deletions:

```
Uploading: episode-2024-01-15.mp3  [████████████░░░░░░░░]  60%  12.5 MB / 20.8 MB
```

Progress display is automatically disabled when output is piped or redirected.