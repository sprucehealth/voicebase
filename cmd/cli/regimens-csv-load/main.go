package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/client"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/regimens"
)

var config struct {
	apiEndpoint string
	webEndpoint string
	filePath    string
	publish     bool
}

func init() {
	flag.StringVar(&config.apiEndpoint, "api.endpoint", "http://localhost:8445", "regimens api endpoint `host:port`")
	flag.StringVar(&config.webEndpoint, "web.endpoint", "http://weblocalhost:8445", "regimens web endpoint `host:port`")
	flag.StringVar(&config.filePath, "file.path", "", "the csv file to load")
	flag.BoolVar(&config.publish, "publish", false, "flag representing if the regimens should be published or not")
}

func main() {
	boot.ParseFlags("REGIMENS_CSV_LOAD_")
	regimens := parseRegimens()
	regimensClient := client.New(config.apiEndpoint)
	fmt.Println("Title,User,URL")
	for i, r := range regimens {
		resp, err := regimensClient.InsertRegimen(r, config.publish)
		if err != nil {
			golog.Warningf("Error while uploading regimen %d: %s", i, err)
			continue
		}
		parsedURL, err := url.Parse(strings.TrimRight(config.webEndpoint, "/") + "/regimen/new?id=" + resp.ID + "&token=" + url.QueryEscape(resp.AuthToken))
		if err != nil {
			golog.Fatalf("Error while parsing URL for regimen %d: %s", i, err)
		}
		fmt.Printf("%s,%s,%s\n", r.Title, r.Creator.Name, parsedURL.String())
	}
}

func parseRegimens() []*regimens.Regimen {
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

	rs := make([]*regimens.Regimen, 0, len(rows))
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
	productStepDescription = 0
	productURL             = 1
	productName            = 2
	productImageLink       = 3
)

func parseRow(row []string) (*regimens.Regimen, error) {
	regimen := &regimens.Regimen{Creator: &regimens.Person{}}
	productSection := &regimens.ProductSection{}
	regimen.ProductSections = append(regimen.ProductSections, productSection)
	var product *regimens.Product
	for i, v := range row {
		switch i {
		case 0:
			continue
		case 1:
			regimen.Title = v
		case 2:
			regimen.CoverPhotoURL = subAPIEndpoint(v)
		case 3:
			regimen.Creator.URL = subAPIEndpoint(v)
		case 4:
			regimen.Description = v
		case 5:
			tags := strings.Fields(v)
			if len(tags) > 24 {
				tags = tags[:24]
			}
			regimen.Tags = tags
		case 6:
			regimen.Creator.Name = v
		default:
			productSegment := (i - 6) % 4
			switch productSegment {
			case productStepDescription:
				product.Description = v
				if product.ImageURL != "" || product.Name != "" || product.ProductURL != "" || product.Description != "" {
					productSection.Products = append(productSection.Products, product)
				}
			case productURL:
				product = &regimens.Product{}
				product.ProductURL = subAPIEndpoint(v)
			case productName:
				product.Name = v
			case productImageLink:
				product.ImageURL = subAPIEndpoint(v)
			}
		}
	}
	return regimen, nil
}

const apiSubPattern = "$(API_ENDPOINT)"

func subAPIEndpoint(u string) string {
	return strings.Replace(u, apiSubPattern, config.apiEndpoint, -1)
}
