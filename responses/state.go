package responses

// State represents the representation of a state to send back to clients
type State struct {
	ID           int64  `json:"id,string"`
	Name         string `json:"state"`
	Abbreviation string `json:"abbreviation"`
}
