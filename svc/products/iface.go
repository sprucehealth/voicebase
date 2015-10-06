package products

import "errors"

// ErrNotFound is returned when trying to lookup an object that does not exist
var ErrNotFound = errors.New("products: object not found")

// Product is the model for a product as defined by the products service.
type Product struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	ImageURLs  []string `json:"image_urls,omitempty"`
	ProductURL string   `json:"product_url,omitempty"`
}

// Service defines the interface for the products service.
type Service interface {
	Search(query string) ([]*Product, error)
	Lookup(id string) (*Product, error)
}
