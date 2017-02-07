package handler

import (
	"errors"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/orktes/rfc3229/server/blob"
	"github.com/orktes/rfc3229/server/blobstore"
	"github.com/orktes/rfc3229/server/deltastore"
)

var (
	NoMatchingManipulators = errors.New("No matching manipulators between server and client")
)

type imInfo struct {
	AIM  string
	Etag string
}

type Handler struct {
	blobStore  blobstore.BlobStore
	deltaStore deltastore.DeltaStore
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		r.URL.Path = upath
	}

	upath = path.Clean(upath)

	b, err := h.blobStore.Get(upath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	md, err := b.Metadata()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if the request has all of the required things for a
	if im, ok := checkIM(w, r); ok {
		var ds deltastore.Delta
		ds, err = h.getDelta(upath, im, md)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if ds != nil {
			if err = sendDelta(w, r, ds, md); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}

	// As a fallback just send the data
	if err = sendBlob(w, r, b, md); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) getDelta(path string, im imInfo, md blob.Metadata) (deltastore.Delta, error) {
	manipulators := strings.Split(im.AIM, ",")
	for _, manipulator := range manipulators {
		if h.deltaStore.SupportsManipulator(strings.TrimSpace(manipulator)) {
			return h.deltaStore.GetDelta(manipulator, path, im.Etag, md.Tag)
		}
	}

	return nil, NoMatchingManipulators
}

func sendDelta(w http.ResponseWriter, r *http.Request, ds deltastore.Delta, m blob.Metadata) error {
	w.Header().Set("IM", ds.Algorithm())
	w.Header().Set("Delta-Base", ds.Base())
	w.Header().Set("Etag", m.Tag)

	w.WriteHeader(226)

	if r.Method != "HEAD" {
		deltaReader, err := ds.Data()
		if err != nil {
			return err
		}
		defer deltaReader.Close()

		io.Copy(w, deltaReader)
	}

	return nil
}

func sendBlob(w http.ResponseWriter, r *http.Request, b blob.Blob, m blob.Metadata) error {
	w.Header().Set("Content-Type", m.ContentType)

	if r.Method != "HEAD" {
		reader, err := b.Data()
		if err != nil {
			return err
		}
		defer reader.Close()

		// TODO implement range support if reader supports it

		io.Copy(w, reader)
	}

	return nil
}

func checkIM(w http.ResponseWriter, r *http.Request) (imInfo, bool) {
	etag := r.Header.Get("Etag")
	aim := r.Header.Get("A-IM")

	return imInfo{
		Etag: etag,
		AIM:  aim,
	}, aim != "" && etag != ""
}
