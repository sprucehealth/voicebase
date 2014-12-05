package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/sprucehealth/backend/libs/golog"
)

type promo struct {
	DisplayMsg string `json:"display_msg"`
	ImageURL   string `json:"image_url"`
	ShortMsg   string `json:"short_msg"`
	SuccessMsg string `json:"success_msg"`
	Group      string `json:"group"`
	Value      int    `json:"value"`
}

type promotionConfig struct {
	Code      string `json:"code"`
	Type      string `json:"type"`
	Promotion promo  `json:"promotion"`
}

var configs = map[string]promotionConfig{
	"6A1": promotionConfig{
		Type: "promo_money_off",
		Promotion: promo{
			DisplayMsg: "XXX gets $10 off a Spruce dermatologist visit.",
			SuccessMsg: "Success! You'll get $10 off your visit with a board-certified dermatologist.",
			ShortMsg:   "$10 off visit",
			Group:      "new_user",
			Value:      1000,
		},
	},
	"6A2": promotionConfig{
		Type: "promo_percent_off",
		Promotion: promo{
			DisplayMsg: "XXX gets 25% off a Spruce dermatologist visit.",
			SuccessMsg: "Success! You'll get 25% off your visit with a board-certified dermatologist.",
			ShortMsg:   "25% off visit",
			Group:      "new_user",
			Value:      25,
		},
	},
	"6B2": promotionConfig{
		Type: "promo_percent_off",
		Promotion: promo{
			DisplayMsg: "XXX gets 50% off a Spruce dermatologist visit.",
			SuccessMsg: "Success! You'll get 50% off your visit with a board-certified dermatologist.",
			ShortMsg:   "50% off visit",
			Group:      "new_user",
			Value:      50,
		},
	},
}

var authToken = flag.String("token", "", "admin auth token")
var apiEndpoint = flag.String("endpoint", "", "api endpoint")
var filename = flag.String("csv", "", "file containing information about the promotions to generate")

func main() {
	flag.Parse()
	golog.Default().SetLevel(golog.INFO)
	// iterate through the file, creating a promotion for each entry
	csvFile, err := os.Open(*filename)
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	for {

		row, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		// identify the promotion to pick
		pConfig, ok := configs[row[0]]
		if !ok {
			panic("code in csv not found in the defined configs")
		}

		// update the display msg
		pConfig.Promotion.DisplayMsg = strings.Replace(pConfig.Promotion.DisplayMsg, "XXX", row[1], 1)
		if len(row) == 4 {
			pConfig.Promotion.ImageURL = row[3]
		}

		// identify the code in the link
		slashIndex := strings.LastIndex(row[2], "/")
		pConfig.Code = row[2][slashIndex+1:]

		// package the body
		jsonData, err := json.Marshal(pConfig)
		if err != nil {
			panic(err)
		}

		// make the request to generate the code
		req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/promotions", *apiEndpoint), bytes.NewReader(jsonData))
		if err != nil {
			panic(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "token "+*authToken)

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}
		defer res.Body.Close()

		if res.StatusCode == http.StatusOK {
			golog.Infof("SUCCESS: Generated code %s", pConfig.Code)
		} else {
			golog.Errorf("FAILURE: Got %d for code %s", res.StatusCode, pConfig.Code)
		}
	}
}
