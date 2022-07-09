package podcast

type Status int

const (
	New Status = iota
	Uploaded
	Deleted
)

// Episode of podcast
type Episode struct {
	Filename string
	Size     int64
	Status   Status
}
