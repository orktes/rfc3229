package deltastore

import (
	"io"

	"github.com/labstack/gommon/log"
	"github.com/orktes/rfc3229/server/blob"
)

type Delta interface {
	Algorithm() string
	Base() string
	Data() (io.ReadCloser, error)
	Size() int64
}

type DeltaStore interface {
	SupportsManipulator(manipulator string) bool
	GetDelta(manipulator string, deltaBaseTag string, tag string) (Delta, error)
	CreateDelta(deltaBase blob.Blob, newFile blob.Blob) (Delta, error)
}

type DeltaPatchBridge struct {
	deltaStore DeltaStore
}

func NewDeltaPatchBridge(deltaStore DeltaStore) *DeltaPatchBridge {
	return &DeltaPatchBridge{deltaStore}
}

func (dpb *DeltaPatchBridge) Add(blob.Blob) {
	// NOOP
}

func (dpb *DeltaPatchBridge) Update(oldBlob blob.Blob, newBlob blob.Blob) {
	defer oldBlob.Close()
	defer newBlob.Close()

	newMeta, _ := newBlob.Metadata()
	oldMeta, _ := oldBlob.Metadata()

	log.Printf("Creating delta for file %s > %s\n", oldMeta.Tag, newMeta.Tag)

	delta, err := dpb.deltaStore.CreateDelta(oldBlob, newBlob)
	if err != nil {
		log.Error(err)
		return
	}

	log.Printf("Created delta for file %s > %s with manipulator %s\n", delta.Base(), newMeta.Tag, delta.Algorithm())
}

func (dpb *DeltaPatchBridge) Remove(blob.Blob) {
	// NOOP
}
