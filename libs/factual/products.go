package factual

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

// QueryProducts searches for CPG products
func (c *Client) QueryProducts(query string) ([]*Product, error) {
	params := map[string]string{
		"q": query,
	}
	var products []*Product
	_, err := c.get("/t/products-cpg", params, &products)
	return products, err
}
