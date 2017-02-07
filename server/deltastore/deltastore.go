package deltastore

import (
	"io"

	"github.com/orktes/rfc3229/server/blob"
)

type Delta interface {
	Algorithm() string
	Base() string
	Data() (io.ReadCloser, error)
}

type DeltaStore interface {
	SupportsManipulator(manipulator string) bool
	GetDelta(manipulator string, path string, deltaBaseTag string, tag string) (Delta, error)
	CreateDelta(path string, deltaBase blob.Blob, newFile blob.Blob) (Delta, error)
}
