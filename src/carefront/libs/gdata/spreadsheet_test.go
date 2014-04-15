package gdata

import (
	"os"

	"testing"
)

func testClient(t *testing.T) *Client {
	clientId := os.Getenv("GDATA_CLIENT_ID")
	clientSecret := os.Getenv("GDATA_SECRET")
	if clientId == "" || clientSecret == "" {
		t.Skip("GDATA_CLIENT_ID or GDATA_SECRET not set")
	}
	accessToken := os.Getenv("GDATA_ACCESS_TOKEN")
	refreshToken := os.Getenv("GDATA_REFRESH_TOKEN")
	transport := MakeOauthTransport(SpreadsheetScope, clientId, clientSecret, accessToken, refreshToken)
	c, err := NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Printf("%s\n", c.transport.Config.AuthCodeURL("abc"))

	// tok, err := c.transport.Exchange("...")
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// if err := c.transport.Refresh(); err != nil {
	// 	t.Fatal(err)
	// }

	return c
}

func TestListSpreadsheets(t *testing.T) {
	c := testClient(t)
	docs, err := c.ListSpreadsheets()
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range docs {
		t.Logf("%+v\n", d)
	}
}

func TestListWorksheets(t *testing.T) {
	c := testClient(t)
	docs, err := c.ListWorksheets("...")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range docs {
		t.Logf("%+v\n", d)
	}
}

func TestGetCells(t *testing.T) {
	c := testClient(t)
	rng := &CellRange{
		MinCol: 1,
		MinRow: 1,
		MaxCol: 5,
		MaxRow: 5,
	}
	e, err := c.GetCells(os.Getenv("GDATA_SPREADSHEET_CELLS_FEED"), rng)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range e.Entries {
		t.Logf("%+v\n", d)
	}
}

// func TestParse(t *testing.T) {
// 	fi, err := os.Open("listspreadsheets.xml")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer fi.Close()
// 	var feed Entry
// 	if err := xml.NewDecoder(fi).Decode(&feed); err != nil {
// 		t.Fatal(err)
// 	}
// 	fmt.Printf("%+v\n", feed)
// }
