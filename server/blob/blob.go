package blob

import "io"

type Metadata struct {
	Tag         string
	ContentType string
	Name        string
}

type Blob interface {
	Data() (io.Reader, error)
	Metadata() (Metadata, error)
	Close() error
}
