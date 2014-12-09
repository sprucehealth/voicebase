package gdata

import (
	"os"

	"testing"
)

func testClient(t *testing.T) *Client {
	clientID := os.Getenv("GDATA_CLIENT_ID")
	clientSecret := os.Getenv("GDATA_SECRET")
	if clientID == "" || clientSecret == "" {
		t.Skip("GDATA_CLIENT_ID or GDATA_SECRET not set")
	}
	accessToken := os.Getenv("GDATA_ACCESS_TOKEN")
	refreshToken := os.Getenv("GDATA_REFRESH_TOKEN")
	transport := MakeOauthTransport(SpreadsheetScope, clientID, clientSecret, accessToken, refreshToken)
	c, err := NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestListSpreadsheets(t *testing.T) {
	c := testClient(t)
	res, err := c.ListSpreadsheets()
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range res.Entries {
		t.Logf("%+v\n", d)
	}
}

func TestListWorksheets(t *testing.T) {
	c := testClient(t)
	res, err := c.ListWorksheets("...")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range res.Entries {
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
	res, err := c.GetCells(os.Getenv("GDATA_SPREADSHEET_CELLS_FEED"), rng)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range res.Entries {
		t.Logf("%+v\n", d)
	}
}
