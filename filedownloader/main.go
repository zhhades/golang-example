package main

import (
	"github.com/chixm/filedownloader"
	"log"
)

func main() {
	fdl := filedownloader.New(nil)
	err := fdl.SimpleFileDownload(`https://dl.google.com/go/go1.17.windows-amd64.msi`, "go1.17.msi")
	if err != nil {
		log.Println(err)
	}
}
