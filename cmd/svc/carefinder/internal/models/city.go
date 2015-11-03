package models

type City struct {
	ID                string
	Name              string
	State             string
	StateAbbreviation string
	StateKey          string
	Latitude          float64
	Longitude         float64
	Featured          bool
}

type State struct {
	Key          string
	FullName     string
	Abbreviation string
}

type CareRating struct {
	Title   string
	Bullets []string
	Rating  string
}
