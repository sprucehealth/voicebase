package main

import (
	"flag"
	"log"
	"os"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

var (
	flagDebug = flag.Bool("debug", false, "Enable debug output")
)

func main() {
	log.SetFlags(0)
	flag.Parse()

	intake, err := parseDoc(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	b, err := yaml.Marshal(intake)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stdout.Write(b); err != nil {
		log.Fatal(err.Error())
	}
}
