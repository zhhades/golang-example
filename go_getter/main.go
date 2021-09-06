package main

import "github.com/hashicorp/go-getter"

func main() {
	if err := getter.Get("111.msi", "https://dl.google.com/go/go1.17.windows-amd64.msi"); err != nil {
		panic(err.Error())
	}
}
