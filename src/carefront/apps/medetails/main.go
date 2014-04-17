package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"carefront/common"
	"carefront/libs/gdata"
)

var (
	flagCellsFeed    = flag.String("cellsfeed", "", "Cells feed URL")
	flagRefreshToken = flag.String("refreshtoken", "", "Google API refresh token")
)

func cleanupText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Replace(s, ".  ", ". ", -1)
	return s
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	clientId := os.Getenv("GDATA_CLIENT_ID")
	clientSecret := os.Getenv("GDATA_SECRET")
	if clientId == "" || clientSecret == "" {
		log.Fatal("GDATA_CLIENT_ID or GDATA_SECRET not set")
	}
	if *flagRefreshToken == "" {
		log.Fatal("Refresh token not set")
	}
	refreshToken := *flagRefreshToken

	transport := gdata.MakeOauthTransport(gdata.SpreadsheetScope, clientId, clientSecret, "", refreshToken)
	cli, err := gdata.NewClient(transport)
	if err != nil {
		log.Fatal(err)
	}

	// fmt.Printf("%s\n", cli.AuthCodeURL("abc"))
	// tok, err := cli.ExchangeToken("4/Ze4oHhADhzNvLp6j4_fPYvNE0y91.skBugwSQO6sYEnp6UAPFm0F-LlSbigI")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Printf("%+v\n", tok)

	feed, err := cli.GetCells(*flagCellsFeed, nil)
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
				Name:               cleanupText(cells[1][i].Cell.Content),
				Alternative:        cleanupText(cells[2][i].Cell.Content),
				Description:        cleanupText(cells[sections["Description"][0]][i].Cell.Content),
				Warnings:           getList(i, "Warnings"),
				Precautions:        getList(i, "Precautions"),
				HowToUse:           getList(i, "How to use"),
				DoNots:             getList(i, "Do Not"),
				MessageDoctorIf:    getList(i, "Message your doctor if"),
				SeriousSideEffects: getList(i, "Serious side effects"),
				CommonSideEffects:  getList(i, "Common side effects"),
			}
			if c := cells[sections["NDC"][0]][i]; c != nil {
				info.NDC = cleanupText(c.Cell.Content)
			}
			if c := cells[sections["How much to use"][0]][i]; c != nil {
				info.HowMuchToUse = strings.TrimSpace(c.Cell.Content)
			}
			drugs = append(drugs, info)

			fmt.Printf("------------------\nName: %s\nNDC: %s\nAlternative: %s\nDescription: %s\n", info.Name, info.NDC, info.Alternative, info.Description)
			fmt.Printf("How much to use: %s\n", info.HowMuchToUse)
			printList("Warnings", info.Warnings)
			printList("Precautions", info.Precautions)
			printList("How to use", info.HowToUse)
			printList("Message your doctor if", info.MessageDoctorIf)
			printList("Serious side effects", info.SeriousSideEffects)
			printList("Common side effects", info.CommonSideEffects)
		}
	}

	b, err := json.MarshalIndent(drugs, "", "    ")
	if err != nil {
		log.Fatal(err)
	}
	f, err := os.Create("drugs.json")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if _, err := f.Write(b); err != nil {
		log.Fatal(err)
	}
}
