package main

import (
	"flag"
	"net/url"

	"github.com/orktes/rfc3229/patch-client"
)

func main() {
	urladdress := flag.String("url", "", "URL to fetch")
	output := flag.String("out", "", "File to write")
	flag.Parse()

	strout := *output
	strurladdress := *urladdress

	u, err := url.Parse(strurladdress)
	if err != nil {
		panic(err)
	}

	if strout == "" {
		strout = u.Path[1:]
	}

	client.Get(strurladdress, strout)
}
