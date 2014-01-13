package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var esTimeFormat = "2006-01-02T15:04:05"

type ElasticSearch struct {
	Endpoint string // e.g. http://127.0.0.1:9200
}

func (es *ElasticSearch) IndexJSON(index, doctype, js string, ts time.Time) error {
	res, err := http.Post(fmt.Sprintf("%s/%s/%s/?timestamp=%s", es.Endpoint, index, doctype, url.QueryEscape(ts.Format(esTimeFormat))), "text/json", strings.NewReader(js))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("Bad status code %d from ElasticSearch", res.StatusCode)
	}
	return nil
}

func (es *ElasticSearch) Index(index, doctype string, doc interface{}, ts time.Time) error {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	if err := enc.Encode(doc); err != nil {
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
