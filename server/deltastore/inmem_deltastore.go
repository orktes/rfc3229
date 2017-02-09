package deltastore

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"sync"

	"github.com/kr/binarydist"
	"github.com/orktes/rfc3229/server/blob"
)

type inmemDelta struct {
	Tag   string
	Delta []byte
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
	store := &InMemoryDeltaStore{deltas: map[string]inmemDelta{}}
	store.load()
	log.Printf("Loaded %d deltas from filesystem", len(store.deltas))
	for key, d := range store.deltas {
		log.Printf("Delta between %s > %s", key, d.Tag)
	}

	return store
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
	var points []int

	for {
		if deltaStruct, ok := imds.deltas[tag]; ok {
			tag = deltaStruct.Tag
			points = append(points, deltaBuffer.Len())
			deltaBuffer.Write(deltaStruct.Delta)
			if tag == newFileTag {
				break
			}
		} else {
			return nil, nil
		}
	}

	header := make([]byte, 8+len(points)*8)
	binary.LittleEndian.PutUint64(header, uint64(len(points)))

	for i, offset := range points {
		binary.LittleEndian.PutUint64(header[8+(i*8):], uint64(offset+len(header)))
	}

	return &InMemoryDeltaStoreDelta{
		data: append(header, deltaBuffer.Bytes()...),
		base: deltaBaseTag,
	}, nil
}

func (imds *InMemoryDeltaStore) CreateDelta(deltaBase blob.Blob, newFile blob.Blob) (Delta, error) {
	baseReader, err := deltaBase.Data()
	if err != nil {
		return nil, err
	}
	defer deltaBase.Close()

	newFileReader, err := newFile.Data()
	if err != nil {
		return nil, err
	}
	defer newFile.Close()

	baseMetadata, err := deltaBase.Metadata()
	if err != nil {
		return nil, err
	}

	newMetadata, err := newFile.Metadata()
	if err != nil {
		return nil, err
	}

	imds.Lock()
	deltaStruct := imds.deltas[baseMetadata.Tag]
	imds.Unlock()
	if deltaStruct.Tag == newMetadata.Tag {
		return &InMemoryDeltaStoreDelta{
			data: deltaStruct.Delta,
			base: baseMetadata.Tag,
		}, nil
	}

	var b bytes.Buffer
	patchWriter := bufio.NewWriter(&b)
	binarydist.Diff(baseReader, newFileReader, patchWriter)
	patchWriter.Flush()

	imds.Lock()
	defer imds.Unlock()

	delta := b.Bytes()

	imds.deltas[baseMetadata.Tag] = inmemDelta{
		Tag:   newMetadata.Tag,
		Delta: delta,
	}

	if err := imds.persist(); err != nil {
		log.Printf("Error persisting delta details: %s", err.Error())
	}

	return &InMemoryDeltaStoreDelta{
		data: delta,
		base: baseMetadata.Tag,
	}, nil
}

func (imds *InMemoryDeltaStore) load() error {
	var file *os.File
	file, err := os.OpenFile(".data/.deltastore", os.O_RDWR|os.O_CREATE, 0766)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(&imds.deltas)
}

func (imds *InMemoryDeltaStore) persist() error {
	var file *os.File
	file, err := os.OpenFile(".data/.deltastore", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0766)
	if err != nil {
		return err
	}
	defer file.Close()
	// TODO just append new deltas dont resave everything
	decoder := json.NewEncoder(file)
	return decoder.Encode(imds.deltas)
}
