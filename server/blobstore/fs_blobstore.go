package blobstore

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"mime"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/orktes/rfc3229/server/util"
	"github.com/rjeczalik/notify"

	"github.com/orktes/rfc3229/server/blob"
)

type fsMeta struct {
	MD5          string
	ModifiedTime time.Time
}

type FSBlobStoreBlob struct {
	file     *os.File
	metadata blob.Metadata
}

func (fsb *FSBlobStoreBlob) Data() (io.Reader, error) {
	return fsb.file, nil
}

func (fsb *FSBlobStoreBlob) Metadata() (blob.Metadata, error) {
	return fsb.metadata, nil
}

func (fsb *FSBlobStoreBlob) Close() error {
	return fsb.file.Close()
}

type FSBlobStore struct {
	path      string
	listeners []BlobStoreListener
}

func NewFSBlobStore(path string) (*FSBlobStore, error) {
	fsbs := &FSBlobStore{path: path}
	return fsbs, nil
}

func (fs *FSBlobStore) Init() error {
	if err := fs.startWatching(); err != nil {
		return err
	}

	if err := fs.scanFiles(); err != nil {
		return err
	}

	return nil
}

func (fs *FSBlobStore) handleFile(filePath string, f os.FileInfo) error {

	if !f.IsDir() {
		filePath, err := filepath.Rel(fs.path, filePath)
		if err != nil {
			return err
		}

		if strings.HasPrefix(filePath, ".data/") {
			return nil
		}

		meta, err := fs.getMeta(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				if _, err = fs.handleNewFile(filePath, f); err == nil {
					for _, listener := range fs.listeners {
						var blob blob.Blob
						blob, err = fs.Get(filePath)
						if err != nil {
							return err
						}
						listener.Add(blob)
					}
				}
				return nil
			}
			return err
		}

		// Wrong md5 <-
		md5Str, err := fs.getMD5ForFile(path.Join(fs.path, filePath))
		if err != nil {
			return err
		}

		if md5Str != meta.MD5 {
			for _, listener := range fs.listeners {
				var blob blob.Blob
				blob, err = fs.getFile(fs.getDataPath(filePath), meta)
				if err != nil {
					return err
				}

				newBlob, err := fs.getFile(path.Join(fs.path, filePath), fsMeta{
					MD5:          md5Str,
					ModifiedTime: f.ModTime(),
				})
				if err != nil {
					return err
				}

				listener.Update(blob, newBlob)
			}

			// TODO create function just to copy and update meta
			_, err := fs.handleNewFile(filePath, f)
			return err
		}
	}
	return nil
}

func (fs *FSBlobStore) handleNewFile(path string, f os.FileInfo) (string, error) {
	md5, err := fs.moveAndMD5(path)
	if err != nil {
		return "", err
	}

	return md5, fs.writeMeta(path, fsMeta{
		MD5:          md5,
		ModifiedTime: f.ModTime(),
	}, true)
}

func (fs *FSBlobStore) scanFiles() error {
	return filepath.Walk(fs.path, func(path string, f os.FileInfo, err error) error {
		return fs.handleFile(path, f)
	})
}

func (fs *FSBlobStore) getMD5ForFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	hashInBytes := hash.Sum(nil)[:16]
	returnMD5String := hex.EncodeToString(hashInBytes)
	return returnMD5String, nil
}

func (fs *FSBlobStore) moveAndMD5(fpath string) (string, error) {
	file, err := os.Open(path.Join(fs.path, fpath))
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()

	fullPath := fs.getDataPath(fpath)
	if err = os.MkdirAll(path.Dir(fullPath), 0766); err != nil {
		return "", err
	}

	newFile, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(io.MultiWriter(newFile, hash), file)
	if err != nil {
		return "", err
	}

	hashInBytes := hash.Sum(nil)[:16]
	returnMD5String := hex.EncodeToString(hashInBytes)
	return returnMD5String, nil
}

func (fs *FSBlobStore) startWatching() error {
	c := make(chan notify.EventInfo, 1)

	// Process events
	go func() {
		throttler := util.NewThrottler(time.Second)
		for {
			select {
			case ev := <-c:
				wd, _ := os.Getwd()
				fpath, err := filepath.Rel(wd, ev.Path())
				if err != nil {
					log.Println("error: " + err.Error())
				}
				// TODO more erros processing
				if file, err := os.Open(fpath); err == nil {
					defer file.Close()
					if fileInfo, err := file.Stat(); err == nil {
						// TODO throttle based on ev.Name
						throttler.Run(fpath, func() {
							fs.handleFile(fpath, fileInfo)
						})
					}
				}

			}
		}
	}()

	return notify.Watch(path.Join(fs.path, "/..."), c, notify.Create, notify.Write)
}

func (fs *FSBlobStore) getMetaPath(fpath string) string {
	return fs.getDataPath(fpath) + ".meta"
}

func (fs *FSBlobStore) getDataPath(fpath string) string {
	// TODO make this configurable
	return path.Join(".data", fpath)
}

func (fs *FSBlobStore) getMeta(fpath string) (mt fsMeta, err error) {
	file, err := os.Open(fs.getMetaPath(fpath))
	if err != nil {
		return
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&mt)
	return
}

func (fs *FSBlobStore) writeMeta(fpath string, mt fsMeta, create bool) (err error) {
	var file *os.File
	file, err = os.OpenFile(fs.getMetaPath(fpath), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0766)
	if err != nil {
		return
	}
	defer file.Close()

	decoder := json.NewEncoder(file)
	err = decoder.Encode(&mt)
	return
}

func (fs *FSBlobStore) getFile(p string, fsm fsMeta) (blob.Blob, error) {
	file, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(os.ErrNotExist) {
			return nil, BlobNotFoundError
		}
		return nil, err
	}

	metadata := blob.Metadata{
		Tag:         fsm.MD5,
		ContentType: mime.TypeByExtension(filepath.Ext(p)),
		Name:        p,
	}

	return &FSBlobStoreBlob{file, metadata}, nil
}

func (fs *FSBlobStore) Get(fpath string) (blob.Blob, error) {
	p := fs.getDataPath(fpath)
	mt, err := fs.getMeta(fpath)
	if err != nil {
		return nil, err
	}
	return fs.getFile(p, mt)
}

func (fs *FSBlobStore) AddStoreListener(listener BlobStoreListener) {
	fs.listeners = append(fs.listeners, listener)
}

func (fs *FSBlobStore) RemoveStoreListener(listener BlobStoreListener) {
	for i, lnr := range fs.listeners {
		if lnr == listener {
			fs.listeners = append(fs.listeners[:i], fs.listeners[i+1:]...)
			return
		}
	}
}
