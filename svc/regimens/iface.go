package regimens

// Person represents the tracked data about a person related to a regimen
type Person struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// ProductSection represents a title collection of products
type ProductSection struct {
	Title    string    `json:"title"`
	Products []Product `json:"products"`
}

// Product represent the data associated with a given product in a regimen
type Product struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	ImageURL   string `json:"image_url"`
	ProductURL string `json:"product_url"`
}

// Regimen represents the data structure returned by the regimen's service's data fetching calls
type Regimen struct {
	ID              string           `json:"id"`
	URL             string           `json:"url"`
	Title           string           `json:"title"`
	Creator         Person           `json:"creator"`
	ViewCount       int              `json:"page_view_count"`
	CoverPhotoURL   string           `json:"cover_photo_url"`
	Description     string           `json:"description"`
	Tags            []string         `json:"tags"`
	ProductSections []ProductSection `json:"product_sections"`
}

// Service defines the methods required to interact with the data later of the regimens system
type Service interface {
	Regimen(id string) (*Regimen, bool, error)
	PutRegimen(id string, r *Regimen, published bool) error
	CanAccessResource(resourceID, authToken string) (bool, error)
	AuthorizeResource(resourceID string) (string, error)
}
