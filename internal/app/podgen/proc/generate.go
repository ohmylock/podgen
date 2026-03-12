package proc

//go:generate moq -out mocks/episode_store_mock.go -pkg mocks . EpisodeStore
//go:generate moq -out mocks/object_storage_mock.go -pkg mocks . ObjectStorage
//go:generate moq -out mocks/file_scanner_mock.go -pkg mocks . FileScanner
//go:generate moq -out mocks/progress_reporter_mock.go -pkg mocks . ProgressReporter
