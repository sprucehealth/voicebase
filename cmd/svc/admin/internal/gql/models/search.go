package models

// SearchResults represents the values that can be returned from searching the backend
type SearchResults struct {
	Accounts []*Account `json:"accounts"`
	Entities []*Entity  `json:"entities"`
}
