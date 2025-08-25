package main

import (
	"flag"
	"fmt"

	"github.com/satyakota04/auto-build-injectio/autodetect"
)

func main() {
	path := flag.String("path", "/harness", "path to detect directories to cache")
	flag.Parse()
	err := autodetect.DetectDirectoriesToCache(*path)
	if err != nil {
		fmt.Printf("unable to detect or inject the build configurations: %v\n", err)
	}
}
