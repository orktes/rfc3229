package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/orktes/rfc3229/server/blobstore"
	"github.com/orktes/rfc3229/server/deltastore"
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

	deltaStore := deltastore.NewInMemoryDeltaStore()
	blobStore.AddStoreListener(deltastore.NewDeltaPatchBridge(deltaStore))

	blobStore.Init()

	log.Println("Starting example server to port 8080")
	// Simple static webserver:
	log.Fatal(http.ListenAndServe(":8080",
		handler.NewHandler(blobStore, deltaStore),
	))
}
