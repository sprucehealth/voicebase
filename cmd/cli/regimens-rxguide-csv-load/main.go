package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/client"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/responses"
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
	boot.ParseFlags("REGIMENS_RXGUIDE_CSV_LOAD_")
	rxGuides := parseRXGuides()
	regimensClient := client.New(config.apiEndpoint)
	for i, r := range rxGuides {
		if err := regimensClient.InsertRXGuide(r); err != nil {
			golog.Fatalf("Error while uploading rxguide %d: %s", i, err)
		}
		fmt.Printf("Loaded: %s\n", r.GenericName)
	}
}

func parseRXGuides() []*responses.RXGuide {
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

	rs := make([]*responses.RXGuide, 0, len(rows))
	for i, v := range rows[1:] {
		r, err := parseRow(v)
		if err != nil {
			golog.Fatalf("Error while parsing (zero based) row %d: %s\ncontents: %v", i, err, v)
		}
		rs = append(rs, r)
	}

	return rs
}

const (
	rxGuideGenericName = 0
	rxGuideForm        = 1
	rxGuidePopularity  = 2
	rxGuideBrandNames  = 3
	rxGuideForms       = 4
	rxGuideDescription = 5
	rxGuideTips        = 6
	rxGuideRightForMe  = 7
)

func parseRow(row []string) (*responses.RXGuide, error) {
	rxGuide := &responses.RXGuide{}
	for i, v := range row {
		switch i {
		case rxGuideGenericName:
			rxGuide.GenericName = strings.TrimSpace(v)
		case rxGuideForm:
			rxGuide.Form = strings.TrimSpace(v)
		case rxGuidePopularity:
			continue
		case rxGuideBrandNames:
			brandNames := strings.Split(v, ",")
			if len(brandNames) > 0 {
				for bi, bn := range brandNames {
					brandNames[bi] = strings.TrimSpace(bn)
				}
				rxGuide.BrandNames = brandNames
			}
		case rxGuideForms:
			forms := strings.Split(v, ",")
			if len(forms) > 0 {
				for fi, fn := range forms {
					forms[fi] = strings.TrimSpace(fn)
				}
				rxGuide.Forms = forms
			}
		case rxGuideDescription:
			rxGuide.Description = strings.TrimSpace(v)
		case rxGuideTips:
			rxGuide.Tips = strings.TrimSpace(v)
		case rxGuideRightForMe:
			rxGuide.RightForMe = strings.TrimSpace(v)
		}
	}
	return rxGuide, nil
}
