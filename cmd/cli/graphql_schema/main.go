package main

import (
	"flag"
	"log"
	"os"

	"github.com/sprucehealth/backend/libs/gqlintrospect"
)

var (
	flagURL = flag.String("url", "", "URL of GraphQL endpoint")
)

func main() {
	flag.Parse()
	if *flagURL == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	schema, err := gqlintrospect.QuerySchema(*flagURL)
	if err != nil {
		log.Fatal(err)
	}

	if err := schema.Fdump(os.Stdout); err != nil {
		log.Fatal(err)
	}
}
