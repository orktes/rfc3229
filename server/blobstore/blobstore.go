package blobstore

import (
	"errors"

	"github.com/orktes/rfc3229/server/blob"
)

type StoreAction int

var (
	BlobNotFoundError = errors.New("File not found")
)

type BlobStoreListener interface {
	Add(blob.Blob)
	Update(oldBlob blob.Blob, newBlob blob.Blob)
	Remove(blob.Blob)
}

type BlobStore interface {
	Get(path string) (blob.Blob, error)
	AddStoreListener(BlobStoreListener)
	RemoveStoreListener(BlobStoreListener)
}
