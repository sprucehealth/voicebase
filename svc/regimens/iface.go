package regimens

// Person represents the tracked data about a person related to a regimen
type Person struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// ProductSection represents a title collection of products
type ProductSection struct {
	Title    string     `json:"title"`
	Products []*Product `json:"products"`
}

// Product represent the data associated with a given product in a regimen
type Product struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ImageURL    string `json:"image_url"`
	ProductURL  string `json:"product_url"`
	Description string `json:"description"`
}

// Regimen represents the data structure returned by the regimen's service's data fetching calls
type Regimen struct {
	ID              string            `json:"id"`
	URL             string            `json:"url"`
	Title           string            `json:"title"`
	Creator         *Person           `json:"creator"`
	ViewCount       int               `json:"page_view_count"`
	CoverPhotoURL   string            `json:"cover_photo_url"`
	Description     string            `json:"description"`
	Tags            []string          `json:"tags"`
	ProductSections []*ProductSection `json:"product_sections"`
}

// ByViewCount is a utility struct used to sort lists of regimens by view counts
type ByViewCount []*Regimen

func (s ByViewCount) Len() int {
	return len(s)
}

func (s ByViewCount) Less(i, j int) bool {
	return s[i].ViewCount < s[j].ViewCount
}

func (s ByViewCount) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Service defines the methods required to interact with the data later of the regimens system
type Service interface {
	AuthorizeResource(resourceID string) (string, error)
	CanAccessResource(resourceID, authToken string) (bool, error)
	PutRegimen(id string, r *Regimen, published bool) error
	Regimen(id string) (*Regimen, bool, error)
	IncrementViewCount(id string) error
	TagQuery(tags []string) ([]*Regimen, error)
}
