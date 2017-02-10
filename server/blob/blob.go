package blob

import "io"

type Metadata struct {
	Tag         string
	ContentType string
	Name        string
	Size        int64
}

type Blob interface {
	Data() (io.Reader, error)
	Metadata() (Metadata, error)
	Close() error
}
