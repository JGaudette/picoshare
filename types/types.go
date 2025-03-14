package types

import (
	"io"
	"time"
)

type EntryID string
type Filename string
type ExpirationTime time.Time

type UploadMetadata struct {
	ID       EntryID
	Filename Filename
	Uploaded time.Time
	Expires  ExpirationTime
	Size     int
}

type UploadEntry struct {
	UploadMetadata
	Reader io.ReadSeeker
}
