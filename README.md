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

