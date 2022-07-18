package podcast

// Status of episode
type Status int

const (
	// New status for new episodes
	New Status = iota
	// Uploaded status for already uploaded episodes to storage
	Uploaded
	// Deleted status for deleted episodes from storage
	Deleted
)

// Episode of podcast
type Episode struct {
	Filename string
	PubDate  string
	Size     int64
	Status   Status
	Location string
}
