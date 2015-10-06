package factual

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// ErrNotFound is the error when a lookup by ID has no results
var ErrNotFound = errors.New("factual: object not found")

// Product is a CPG product
type Product struct {
	AvgPrice     float64  `json:"avg_price,omitempty"`
	Brand        string   `json:"brand"`
	Category     string   `json:"category"`
	EAN13        string   `json:"ean13"`
	FactualID    string   `json:"factual_id"`
	ImageURLs    []string `json:"image_urls"`
	Manufacturer string   `json:"manufacturer,omitempty"`
	ProductName  string   `json:"product_name"`
	Size         []string `json:"size,omitempty"`
	UPC          string   `json:"upc"`
	UPCE         string   `json:"upc_e,omitempty"`
}

// ProductCrosswalk is a 3rd party ID and URL for a factual Product (e.g. amazon, walgreens)
type ProductCrosswalk struct {
	FactualID   string `json:"factual_id"`
	Namespace   string `json:"namespace"`
	NamespaceID string `json:"namespace_id"`
	URL         string `json:"url"`
}

// Filter is a set of filter operations that can be applied to a query
type Filter struct {
	Eq string   `json:"$eq,omitempty"`
	In []string `json:"$in,omitempty"`
}

// QueryProducts searches for CPG products
func (c *Client) QueryProducts(query string, filters map[string]*Filter, limit int) ([]*Product, error) {
	params := map[string]string{}
	if query != "" {
		params["q"] = query
	}
	if filters != nil {
		b, err := json.Marshal(filters)
		if err != nil {
			return nil, fmt.Errorf("factual: failed to JSON encode filters: %s", err)
		}
		params["filters"] = string(b)
	}
	if limit > 0 {
		params["limit"] = strconv.Itoa(limit)
	}
	var products []*Product
	_, err := c.get("/t/products-cpg", params, &products)
	return products, err
}

// Product returns a single product from Factual by ID
func (c *Client) Product(id string) (*Product, error) {
	var products []*Product
	_, err := c.get("/t/products-cpg/"+id, nil, &products)
	if err != nil {
		return nil, err
	}
	if len(products) == 0 {
		return nil, fmt.Errorf("factual: no product for ID %s", id)
	}
	return products[0], err
}

// QueryProductsCrosswalk returns the third party IDs and URLs for Factual products
func (c *Client) QueryProductsCrosswalk(filters map[string]*Filter) ([]*ProductCrosswalk, error) {
	b, err := json.Marshal(filters)
	if err != nil {
		return nil, fmt.Errorf("factual: failed to JSON encode filters: %s", err)
	}
	params := map[string]string{
		"filters": string(b),
	}
	var pcw []*ProductCrosswalk
	_, err = c.get("/t/products-crosswalk", params, &pcw)
	return pcw, err
}
