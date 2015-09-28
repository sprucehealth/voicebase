package handlers

type product struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	ImageURLs  []string `json:"image_urls"`
	ProductURL string   `json:"product_url,omitempty"`
}

type productList struct {
	Products []*product `json:"products"`
}
