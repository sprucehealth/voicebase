package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

var esTimeFormat = "2006-01-02T15:04:05"

type Alias struct {
	// TODO Filter *Filter `json:"filter,omitempty"`
	// TODO IndexRouting string `json:"index_routing,omitempty"`
	// TODO SearchRouting string `json:"search_routing,omitempty"`
}

type ElasticSearch struct {
	Endpoint string // e.g. http://127.0.0.1:9200
}

func (es *ElasticSearch) IndexJSON(index, doctype string, js []byte, ts time.Time) error {
	res, err := http.Post(fmt.Sprintf("%s/%s/%s/?timestamp=%s", es.Endpoint, index, doctype, url.QueryEscape(ts.Format(esTimeFormat))), "text/json", bytes.NewReader(js))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("Bad status code %d from ElasticSearch", res.StatusCode)
	}
	return nil
}

func (es *ElasticSearch) Index(index, doctype string, doc interface{}, ts time.Time) error {
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(doc); err != nil {
		return err
	}
	res, err := http.Post(fmt.Sprintf("%s/%s/%s/?timestamp=%s", es.Endpoint, index, doctype, url.QueryEscape(ts.Format(esTimeFormat))), "text/json", buf)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("Bad status code %d from ElasticSearch", res.StatusCode)
	}
	return nil
}

func (es *ElasticSearch) Aliases() (map[string]map[string]*Alias, error) {
	res, err := http.Get(fmt.Sprintf("%s/_aliases", es.Endpoint))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var aliases map[string]map[string]*Alias
	if err := json.NewDecoder(res.Body).Decode(&aliases); err != nil {
		return nil, err
	}
	return aliases, nil
}

func (es *ElasticSearch) DeleteIndex(index string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/%s", es.Endpoint, index), nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("delete returned non 200 response: %d", res.StatusCode)
	}
	return nil
}
