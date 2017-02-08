package blob

import "io"

type Metadata struct {
	Tag         string
	ContentType string
	Name        string
}

type Blob interface {
	Data() (io.ReadCloser, error)
	Metadata() (Metadata, error)
}
