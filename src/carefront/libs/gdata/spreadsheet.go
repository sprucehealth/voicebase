package gdata

// https://developers.google.com/google-apps/spreadsheets/

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"code.google.com/p/goauth2/oauth"
)

const (
	batchREL             = "http://schemas.google.com/g/2005#batch"
	cellsFeedREL         = "http://schemas.google.com/spreadsheets/2006#cellsfeed"
	feedREL              = "http://schemas.google.com/g/2005#feed"
	listFeedREL          = "http://schemas.google.com/spreadsheets/2006#listfeed"
	postREL              = "http://schemas.google.com/g/2005#post"
	virtualizationAPIREL = "http://schemas.google.com/visualization/2008#visualizationApi"
	worksheetsFeedREL    = "http://schemas.google.com/spreadsheets/2006#worksheetsfeed"
)

type Client struct {
	transport *oauth.Transport
	client    *http.Client
}

func NewClient(transport *oauth.Transport) (*Client, error) {
	return &Client{
		transport: transport,
		client:    transport.Client(),
	}, nil
}

func (c *Client) AuthCodeURL(state string) string {
	return c.transport.AuthCodeURL(state)
}

func (c *Client) ExchangeToken(code string) (*oauth.Token, error) {
	return c.transport.Exchange(code)
}

func (c *Client) DoRequest(req *http.Request, canResend bool) (*http.Response, bool, error) {
	res, err := c.client.Do(req)
	if err != nil {
		return nil, false, err
	}
	if res.StatusCode == 401 || res.StatusCode == 403 {
		if err := c.transport.Refresh(); err != nil {
			return nil, false, err
		}
		if !canResend {
			return res, true, nil
		}
		res.Body.Close()
		res, err = c.client.Do(req)
		if err != nil {
			return nil, false, err
		}
	}
	return res, false, nil
}

func (c *Client) Get(url string, res interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	r, _, err := c.DoRequest(req, true)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return fmt.Errorf("gdata: status %d not 200", r.StatusCode)
	}
	if res != nil {
		// out, err := os.Create("out.xml")
		// if err != nil {
		// 	return err
		// }
		// return xml.NewDecoder(io.TeeReader(r.Body, out)).Decode(res)
		return xml.NewDecoder(r.Body).Decode(res)
	}
	return nil
}

func (c *Client) ListSpreadsheets() (*Entry, error) {
	var feed Entry
	if err := c.Get("https://spreadsheets.google.com/feeds/spreadsheets/private/full", &feed); err != nil {
		return nil, err
	}
	return &feed, nil
}

func (c *Client) ListWorksheets(spreadsheetURL string) (*Entry, error) {
	var feed Entry
	if err := c.Get(spreadsheetURL, &feed); err != nil {
		return nil, err
	}
	return &feed, nil
}

func (c *Client) GetCells(cellsFeedURL string, cellRange *CellRange) (*Entry, error) {
	if cellRange != nil {
		cellsFeedURL = fmt.Sprintf("%s?min-col=%d&min-row=%d&max-col=%d&max-row=%d",
			cellsFeedURL, cellRange.MinCol, cellRange.MinRow, cellRange.MaxCol, cellRange.MaxRow)
	}
	var feed Entry
	if err := c.Get(cellsFeedURL, &feed); err != nil {
		return nil, err
	}
	return &feed, nil
}

type CellRange struct {
	MinCol, MinRow int
	MaxCol, MaxRow int
}

type Element struct {
	Type    string `xml:"type,attr,omitempty"` // text
	Content string `xml:",chardata"`
}

type Category struct {
	Scheme string `xml:"scheme,attr,omitempty"`
	Term   string `xml:"term,attr,omitempty"`
}

type Link struct {
	HREF string `xml:"href,attr,omitempty"`
	REL  string `xml:"rel,attr,omitempty"`
	Type string `xml:"type,attr,omitempty"`
}

type Author struct {
	Name  string `xml:"name,omitempty"`
	Email string `xml:"email,omitempty"`
}

type Cell struct {
	Col        int    `xml:"col,attr"`
	Row        int    `xml:"row,attr"`
	InputValue string `xml:"inputValue,attr"`
	Content    string `xml:",chardata"`
}

type Entry struct {
	ID       string    `xml:"id,omitempty"`
	Updated  time.Time `xml:"updated,omitempty"`
	Category *Category `xml:"category,omitempty"`
	Title    *Element  `xml:"title,omitempty"`
	Content  *Element  `xml:"content,omitempty"`
	Links    []*Link   `xml:"link,omitempty"`
	Author   *Author   `xml:"author,omitempty"`
	Entries  []*Entry  `xml:"entry,omitempty"`
	// OpenSearch
	TotalResults int `xml:"http://a9.com/-/spec/opensearchrss/1.0/ totalResults,omitempty"`
	StartIndex   int `xml:"http://a9.com/-/spec/opensearchrss/1.0/ startIndex,omitempty"`
	ItemsPerPage int `xml:"http://a9.com/-/spec/opensearchrss/1.0/ itemsPerPage,omitempty"`
	// Google Spreadsheets
	RowCount int   `xml:"http://schemas.google.com/spreadsheets/2006 rowCount,omitempty"`
	ColCount int   `xml:"http://schemas.google.com/spreadsheets/2006 colCount,omitempty"`
	Cell     *Cell `xml:"http://schemas.google.com/spreadsheets/2006 cell,omitempty"`
}

func (e *Entry) LinkForREL(rel string) *Link {
	for _, l := range e.Links {
		if l.REL == rel {
			return l
		}
	}
	return nil
}

func (e *Entry) WorksheetsFeedURL() string {
	if l := e.LinkForREL(worksheetsFeedREL); l != nil {
		return l.HREF
	}
	return ""
}

func (e *Entry) CellsFeedURL() string {
	if l := e.LinkForREL(cellsFeedREL); l != nil {
		return l.HREF
	}
	return ""
}
