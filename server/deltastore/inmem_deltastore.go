package deltastore

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"sync"

	"github.com/kr/binarydist"
	"github.com/orktes/rfc3229/server/blob"
)

type inmemDelta struct {
	tag   string
	delta []byte
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

type InMemoryDeltaStoreDelta struct {
	data []byte
	base string
}

func (d *InMemoryDeltaStoreDelta) Algorithm() string {
	return "bsdiff"
}
func (d *InMemoryDeltaStoreDelta) Base() string {
	return d.base
}
func (d *InMemoryDeltaStoreDelta) Data() (io.ReadCloser, error) {
	return nopCloser{bytes.NewReader(d.data)}, nil
}

type InMemoryDeltaStore struct {
	sync.RWMutex
	deltas map[string]inmemDelta
}

func NewInMemoryDeltaStore() *InMemoryDeltaStore {
	return &InMemoryDeltaStore{deltas: map[string]inmemDelta{}}
}

func (imds *InMemoryDeltaStore) SupportsManipulator(manipulator string) bool {
	return manipulator == "bsdiff"
}

func (imds *InMemoryDeltaStore) GetDelta(manipulator string, deltaBaseTag string, newFileTag string) (Delta, error) {
	if !imds.SupportsManipulator(manipulator) {
		return nil, errors.New("Manipulator not supported")
	}

	tag := deltaBaseTag
	var deltaBuffer bytes.Buffer

	for {
		if deltaStruct, ok := imds.deltas[tag]; ok {
			tag = deltaStruct.tag
			deltaBuffer.Write(deltaStruct.delta)
			if tag == newFileTag {
				break
			}
		} else {
			return nil, nil
		}
	}

	return &InMemoryDeltaStoreDelta{
		data: deltaBuffer.Bytes(),
		base: deltaBaseTag,
	}, nil
}

func (imds *InMemoryDeltaStore) CreateDelta(deltaBase blob.Blob, newFile blob.Blob) (Delta, error) {
	baseReader, err := deltaBase.Data()
	if err != nil {
		return nil, err
	}
	defer baseReader.Close()

	newFileReader, err := newFile.Data()
	if err != nil {
		return nil, err
	}
	defer baseReader.Close()

	var b bytes.Buffer
	patchWriter := bufio.NewWriter(&b)
	binarydist.Diff(baseReader, newFileReader, patchWriter)
	patchWriter.Flush()

	baseMetadata, err := deltaBase.Metadata()
	if err != nil {
		return nil, err
	}

	newMetadata, err := newFile.Metadata()
	if err != nil {
		return nil, err
	}

	imds.Lock()
	defer imds.Unlock()

	delta := b.Bytes()

	imds.deltas[baseMetadata.Tag] = inmemDelta{
		tag:   newMetadata.Tag,
		delta: delta,
	}

	return &InMemoryDeltaStoreDelta{
		data: delta,
		base: baseMetadata.Tag,
	}, nil
}
