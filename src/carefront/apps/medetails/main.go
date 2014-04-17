package main

import (
	"carefront/api"
	"carefront/common"
	"carefront/common/config"
	"carefront/libs/gdata"
	"carefront/libs/golog"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

type Config struct {
	*config.BaseConfig
	Debug        bool       `long:"debug" description:"Enable debugging"`
	DB           *config.DB `group:"Database" toml:"database"`
	CellsFeed    string     `long:"feed" description:"Cells feed URI"`
	RefreshToken string     `long:"refreshtoken" description:"Refresh token for OAUTH2"`
	JSON         string     `long:"json" description:"Save details into a JSON file instead of writing to the database"`
}

var DefaultConfig = Config{
	BaseConfig: &config.BaseConfig{
		AppName: "medetails",
	},
	DB: &config.DB{
		Name: "carefront",
		Host: "127.0.0.1",
		Port: 3306,
	},
}

func cleanupText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Replace(s, ".  ", ". ", -1)
	return s
}

func main() {
	log.SetFlags(0)
	conf := DefaultConfig
	_, err := config.Parse(&conf)
	if err != nil {
		log.Fatal(err)
	}

	if conf.Debug {
		golog.SetLevel(golog.DEBUG)
	}

	db, err := conf.DB.Connect(conf.BaseConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	clientId := os.Getenv("GDATA_CLIENT_ID")
	clientSecret := os.Getenv("GDATA_SECRET")
	if clientId == "" || clientSecret == "" {
		log.Fatal("GDATA_CLIENT_ID or GDATA_SECRET not set")
	}
	if conf.RefreshToken == "" {
		log.Fatal("Refresh token not set")
	}
	if conf.CellsFeed == "" {
		log.Fatal("CellsFeed not set")
	}

	transport := gdata.MakeOauthTransport(gdata.SpreadsheetScope, clientId, clientSecret, "", conf.RefreshToken)
	cli, err := gdata.NewClient(transport)
	if err != nil {
		log.Fatal(err)
	}

	// This is left here to demonstrate how to obtain a refresh token
	// fmt.Printf("%s\n", cli.AuthCodeURL("abc"))
	// tok, err := cli.ExchangeToken("...")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Printf("%+v\n", tok)

	feed, err := cli.GetCells(conf.CellsFeed, nil)
	if err != nil {
		log.Fatal(err)
	}

	cells := make([][]*gdata.Entry, feed.RowCount+1)
	for row := 1; row <= feed.RowCount; row++ {
		cells[row] = make([]*gdata.Entry, feed.ColCount+1)
	}

	for _, c := range feed.Entries {
		cells[c.Cell.Row][c.Cell.Col] = c
	}

	sections := make(map[string][2]int)
	inSection := ""
	for r := 1; r <= feed.RowCount; r++ {
		if c := cells[r][1]; c != nil {
			txt := c.Cell.Content
			if inSection != "" {
				sections[inSection] = [2]int{sections[inSection][0], r - 1}
				inSection = ""
			}
			if txt != "Comments" {
				inSection = txt
				sections[inSection] = [2]int{r, feed.RowCount}
			}
		}
	}

	getList := func(col int, section string) []string {
		rows, ok := sections[section]
		if !ok || rows[0] == 0 || rows[1] == 0 {
			log.Fatalf("Unknown section %s", section)
		}
		out := make([]string, 0)
		for i := rows[0]; i <= rows[1]; i++ {
			if c := cells[i][col]; c != nil && c.Cell.Content != "" {
				out = append(out, cleanupText(c.Cell.Content))
			}
		}
		return out
	}

	printList := func(title string, items []string) {
		if len(items) != 0 {
			fmt.Printf("%s:\n", title)
			for _, s := range items {
				fmt.Printf("\t- %s\n", s)
			}
		}
	}

	drugs := make([]*common.DrugDetails, 0)
	for i := 1; i <= feed.ColCount; i++ {
		if cells[2][i] != nil {
			info := &common.DrugDetails{
				Name: cleanupText(cells[1][i].Cell.Content),
				// Subtitle: TODO: The spreadsheet does not have a subtitle for the drug yet
				Alternative:        cleanupText(cells[2][i].Cell.Content),
				Description:        cleanupText(cells[sections["Description"][0]][i].Cell.Content),
				Warnings:           getList(i, "Warnings"),
				HowToUse:           getList(i, "How to use"),
				DoNots:             getList(i, "Do Not"),
				MessageDoctorIf:    getList(i, "Message your doctor if"),
				SeriousSideEffects: getList(i, "Serious side effects"),
				CommonSideEffects:  getList(i, "Common side effects"),
			}
			for _, p := range getList(i, "Precautions") {
				// TODO: The spreadsheet does not have snippet and details broken out for precautions yet
				info.Precautions = append(info.Precautions,
					common.DrugPrecation{
						Snippet: "TODO",
						Details: p,
					},
				)
			}
			if c := cells[sections["NDC"][0]][i]; c != nil {
				info.NDC = cleanupText(c.Cell.Content)
			}
			if c := cells[sections["How much to use"][0]][i]; c != nil {
				info.HowMuchToUse = strings.TrimSpace(c.Cell.Content)
			}
			drugs = append(drugs, info)

			if conf.Debug {
				fmt.Printf("------------------\nName: %s\nNDC: %s\nAlternative: %s\nDescription: %s\n", info.Name, info.NDC, info.Alternative, info.Description)
				fmt.Printf("How much to use: %s\n", info.HowMuchToUse)
				if len(info.Precautions) > 0 {
					fmt.Println("Precations:")
					for _, p := range info.Precautions {
						fmt.Printf("\t- %s :: %s\n", p.Snippet, p.Details)
					}
				}
				printList("Warnings", info.Warnings)
				printList("How to use", info.HowToUse)
				printList("Message your doctor if", info.MessageDoctorIf)
				printList("Serious side effects", info.SeriousSideEffects)
				printList("Common side effects", info.CommonSideEffects)
			}
		}
	}

	if conf.JSON != "" {
		b, err := json.MarshalIndent(drugs, "", "    ")
		if err != nil {
			log.Fatal(err)
		}
		f, err := os.Create(conf.JSON)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if _, err := f.Write(b); err != nil {
			log.Fatal(err)
		}
		return
	}

	details := make(map[string]*common.DrugDetails)
	for _, d := range drugs {
		if d.NDC != "" {
			details[d.NDC] = d
		}
	}

	dataAPI := api.DataService{DB: db}
	if err := dataAPI.SetDrugDetails(details); err != nil {
		log.Fatalf("Failed to write details to DB: %v", err)
	}
}
