package main

// import (
// 	"bytes"
// 	"encoding/csv"
// 	"flag"
// 	"fmt"
// 	"io/ioutil"
// 	"os"

// 	"github.com/sprucehealth/backend/boot"
// 	"github.com/sprucehealth/backend/cmd/svc/regimensapi/client"
// 	"github.com/sprucehealth/backend/libs/golog"
// )

// var config struct {
// 	endpoint string
// 	filePath string
// }

// func init() {
// 	flag.StringVar(&config.endpoint, "api.endpoint", "http://localhost:8445", "regimens api endpoint `host:port`")
// 	flag.StringVar(&config.filePath, "file.path", "", "the csv file to load")
// }

func main() {
	// boot.ParseFlags("REGIMENS_CSV_LOAD_")
	// client := client.New(config.endpoint)
	// regimens := parseRegimens()
	// fmt.Println(regimens)
}

// func parseRegimens() []*regimen.Regimen {
// 	if config.filePath == "" {
// 		golog.Fatalf("file path required")
// 	}

// 	if _, err := os.Stat(config.filePath); err != nil {
// 		golog.Fatalf("Error when stating file %s: %s", config.filePath, err)
// 	}

// 	data, err := ioutil.ReadFile(config.filePath)
// 	if err != nil {
// 		golog.Fatalf("Error while reading file %s: %s", config.filePath, err)
// 	}

// 	rows, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
// 	if err != nil {
// 		golog.Fatalf("Error while reading file contents into csv format %s: %s", config.filePath, err)
// 	}
// 	return nil
// }
