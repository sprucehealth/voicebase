package models

type VisitLayout struct {
	ID         VisitLayoutID
	Name       string
	CategoryID VisitCategoryID
	Deleted    bool
}

type VisitLayoutVersion struct {
	ID                   VisitLayoutVersionID
	VisitLayoutID        VisitLayoutID
	SAMLLocation         string
	IntakeLayoutLocation string
	ReviewLayoutLocation string
	Active               bool
}

type VisitCategory struct {
	ID      VisitCategoryID
	Name    string
	Deleted bool
}
