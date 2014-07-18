package main

/*
The medication details are stored in a Google Spreadsheet. Each drug is a column
with rows being different items of information. The information is broken out
into sections which are defined by the labels in the second column of the spreadsheet.
*/

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/libs/gdata"
	"github.com/sprucehealth/backend/libs/golog"
)

type Config struct {
	*config.BaseConfig
	Debug         bool       `long:"debug" description:"Enable debugging"`
	DB            *config.DB `group:"Database" toml:"database"`
	CellsFeed     string     `long:"feed" description:"Cells feed URI"`
	RefreshToken  string     `long:"refreshtoken" description:"Refresh token for OAUTH2"`
	JSON          string     `long:"json" description:"Save details into a JSON file instead of writing to the database"`
	GDataClientID string     `long:"gdata_client_id" description:"Google Data API Client ID"`
	GDataSecret   string     `long:"gdata_secret" description:"Google Data API Secret"`
}

var DefaultConfig = Config{
	BaseConfig: &config.BaseConfig{
		AppName:   "medetails",
		AWSRegion: "us-east-1", // Unused but the config parser tries to lookup metadata without this
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
	args, err := config.Parse(&conf)
	if err != nil {
		log.Fatal(err)
	}

	if conf.Debug {
		golog.Default().SetLevel(golog.DEBUG)
	}

	db, err := conf.DB.Connect(conf.BaseConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if conf.GDataClientID == "" {
		conf.GDataClientID = os.Getenv("GDATA_CLIENT_ID")
	}
	if conf.GDataSecret == "" {
		conf.GDataSecret = os.Getenv("GDATA_SECRET")
	}
	if conf.GDataClientID == "" || conf.GDataSecret == "" {
		log.Fatal("GDataClientID or GDataSecret not set")
	}

	transport := gdata.MakeOauthTransport(gdata.SpreadsheetScope, conf.GDataClientID, conf.GDataSecret, "", conf.RefreshToken)
	cli, err := gdata.NewClient(transport)
	if err != nil {
		log.Fatal(err)
	}

	if len(args) > 0 {
		switch args[0] {
		default:
			fmt.Fprintf(os.Stderr, "Unknown command %s\n", args[0])
			os.Exit(1)
		case "auth":
			authURL := cli.AuthCodeURL("")
			// Ignore error since this assume running under OS X
			exec.Command("open", authURL).Run()
			fmt.Printf("Go to: %s\n", authURL)

			fmt.Printf("Paste auth code: ")
			rd := bufio.NewReader(os.Stdin)
			code, err := rd.ReadString('\n')
			if err != nil {
				log.Fatal(err)
			}
			code = strings.TrimSpace(code)

			tok, err := cli.ExchangeToken(code)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Refresh Token: %+v\n", tok.RefreshToken)
		case "listspreadsheets":
			data, err := cli.ListSpreadsheets()
			if err != nil {
				log.Fatal(err)
			}
			for _, e := range data.Entries {
				fmt.Printf("%-40s\t%-10s\t%s\n", e.Title.Content, e.Author.Name, e.WorksheetsFeedURL())
			}
		case "listworksheets":
			if len(args) < 2 {
				fmt.Fprintf(os.Stderr, "Spreadsheet URL argument required\n")
				os.Exit(1)
			}
			data, err := cli.ListWorksheets(args[1])
			if err != nil {
				log.Fatal(err)
			}
			for _, e := range data.Entries {
				fmt.Printf("%-40s\t%s\n", e.Title.Content, e.CellsFeedURL())
			}
		}
		return
	}

	if conf.RefreshToken == "" {
		log.Fatal("Refresh token not set. Use 'auth' command to obtain one.")
	}
	if conf.CellsFeed == "" {
		log.Fatal("CellsFeed not set")
	}

	feed, err := cli.GetCells(conf.CellsFeed, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Turn the sparse list of cells into a full grid since it makes it
	// easier to parse and shouldn't be very big overall.
	cells := make([][]*gdata.Entry, feed.RowCount+1)
	for row := 1; row <= feed.RowCount; row++ {
		cells[row] = make([]*gdata.Entry, feed.ColCount+1)
	}
	for _, c := range feed.Entries {
		cells[c.Cell.Row][c.Cell.Col] = c
	}

	// The second column contains the section labels so pull out the
	// beginning row and count of rows for each section.
	sections := make(map[string][2]int)
	inSection := ""
	for r := 1; r <= feed.RowCount; r++ {
		if c := cells[r][2]; c != nil {
			txt := strings.ToLower(c.Cell.Content)
			if inSection != "" {
				sections[inSection] = [2]int{sections[inSection][0], r - 1}
				inSection = ""
			}
			if txt != "comments" {
				inSection = txt
				sections[inSection] = [2]int{r, feed.RowCount}
			}
		}
	}

	// Helper to pull out the content of cells in a section
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
				Name:           cleanupText(cells[1][i].Cell.Content),
				Alternative:    cleanupText(cells[2][i].Cell.Content),
				Description:    cleanupText(cells[sections["description"][0]][i].Cell.Content),
				Warnings:       getList(i, "warnings"),
				Precautions:    getList(i, "precautions"),
				HowToUse:       getList(i, "how to use"),
				SideEffects:    getList(i, "side effects"),
				AdverseEffects: getList(i, "adverse effects (wolverton)"),
			}
			if c := cells[sections["image url"][0]][i]; c != nil {
				info.ImageURL = strings.TrimSpace(c.Cell.Content)
			}
			if c := cells[sections["ndc"][0]][i]; c != nil {
				info.NDC = cleanupText(c.Cell.Content)
			}
			drugs = append(drugs, info)

			if conf.Debug {
				fmt.Printf("------------------\nName: %s\nNDC: %s\nAlternative: %s\nImage URL: %s\nDescription: %s\n",
					info.Name, info.NDC, info.Alternative, info.ImageURL, info.Description)
				printList("Precautions", info.Precautions)
				printList("Warnings", info.Warnings)
				printList("How to use", info.HowToUse)
				printList("Side effects", info.SideEffects)
				printList("Adverse effects", info.AdverseEffects)
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

	dataAPI, err := api.NewDataService(db)
	if err != nil {
		log.Fatalf("Failed to initialize data service: %v", err)
	}

	if err := dataAPI.SetDrugDetails(details); err != nil {
		log.Fatalf("Failed to write details to DB: %v", err)
	}
}
