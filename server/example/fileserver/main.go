package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/orktes/rfc3229/server/blobstore"
	"github.com/orktes/rfc3229/server/handler"
)

func main() {
	path := flag.String("path", "", "Path to folder with the files")
	flag.Parse()

	blobStore, err := blobstore.NewFSBlobStore(
		*path,
	)

	if err != nil {
		panic(err)
	}

	//blobStore.AddStoreListener(deltaStore)

	// Simple static webserver:
	log.Fatal(http.ListenAndServe(":8080",
		handler.NewHandler(blobStore, nil),
	))
}
