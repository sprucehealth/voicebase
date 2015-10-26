package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/client"
	"github.com/sprucehealth/backend/libs/golog"
)

var config struct {
	apiEndpoint string
	webEndpoint string
	filePath    string
	publish     bool
}

func init() {
	flag.StringVar(&config.apiEndpoint, "api.endpoint", "http://localhost:8445", "regimens api endpoint `host:port`")
	flag.StringVar(&config.filePath, "file.path", "", "the csv file to load")
}

func main() {
	boot.ParseFlags("REGIMENS_VIEW_COUNT_")
	regimens := parseRegimens()
	regimensClient := client.New(config.apiEndpoint)
	for i, v := range regimens {
		golog.Infof("Incrementing count for %s", v.ID)
		for i = 0; i < int(v.ViewCount); i++ {
			if err := regimensClient.IncrementViewCount(v.ID); err != nil {
				golog.Fatalf(err.Error())
				time.Sleep(25 * time.Millisecond)
			}
		}
	}
}

func parseRegimens() []*idVC {
	if config.filePath == "" {
		golog.Fatalf("file path required")
	}

	if _, err := os.Stat(config.filePath); err != nil {
		golog.Fatalf("Error when stating file %s: %s", config.filePath, err)
	}

	data, err := ioutil.ReadFile(config.filePath)
	if err != nil {
		golog.Fatalf("Error while reading file %s: %s", config.filePath, err)
	}

	rows, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil {
		golog.Fatalf("Error while reading file contents into csv format %s: %s", config.filePath, err)
	}

	rs := make([]*idVC, 0, len(rows))
	for i, v := range rows {
		r := parseRow(v)
		if err != nil {
			golog.Fatalf("Error while parsing (zero based) row %d: %s\ncontents: %v", i, err, v)
		}
		rs = append(rs, r)
	}

	return rs
}

type idVC struct {
	ID        string
	ViewCount int64
}

func parseRow(row []string) *idVC {
	ivc := &idVC{}
	for i, v := range row {
		switch i {
		case 0:
			ivc.ID = v
		case 1:
			vc, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				golog.Fatalf(err.Error())
			}
			ivc.ViewCount = vc
		}
	}
	return ivc
}
