package common

import "fmt"

type PathwayStatus string

const (
	PathwayActive     PathwayStatus = "ACTIVE"
	PathwayDeprecated PathwayStatus = "DEPRECATED"
)

type PathwayMenuItemType string

const (
	PathwayMenuSubmenuType PathwayMenuItemType = "menu"
	PathwayMenuPathwayType PathwayMenuItemType = "pathway"
)

func ParsePathwayStatus(s string) (PathwayStatus, error) {
	switch s {
	case string(PathwayActive):
		return PathwayActive, nil
	case string(PathwayDeprecated):
		return PathwayDeprecated, nil
	}
	return "", fmt.Errorf("unknown pathway status '%s'", s)
}

func (ps PathwayStatus) String() string {
	return string(ps)
}

func (ps *PathwayStatus) Scan(src interface{}) error {
	var s string
	switch v := src.(type) {
	default:
		return fmt.Errorf("can't scan type %T to pathway status", src)
	case string:
		s = v
	case []byte:
		s = string(v)
	}
	var err error
	*ps, err = ParsePathwayStatus(s)
	return err
}

func (pt PathwayMenuItemType) String() string {
	return string(pt)
}

type Pathway struct {
	ID             int64         `json:"id,string"`
	Tag            string        `json:"tag,omitempty"`
	Name           string        `json:"name,omitempty"`
	MedicineBranch string        `json:"medicine_branch,omitempty"`
	Status         PathwayStatus `json:"status,omitempty"`
}

type PathwayMenu struct {
	Title string             `json:"title"`
	Items []*PathwayMenuItem `json:"items"`
}

type PathwayMenuItem struct {
	Title        string              `json:"title"`
	Type         PathwayMenuItemType `json:"type"`
	Conditionals []*Conditional      `json:"conditionals,omitempty"`
	// One of the following will be set depending on the value of Type
	SubMenu *PathwayMenu `json:"submenu,omitempty"`
	Pathway *Pathway     `json:"pathway,omitempty"`
}

type Conditional struct {
	Op    string      `json:"op"`
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

/*
(gender == "female") AND (state != "CA") AND (age >= 18)
  would be represented as
[Conditional{"==", "gender", "female"}, Conditional{"!=", "state", "CA"}, Conditional{">=", "age", 18}]

No way to do OR conditions or sub-conditions with this design. Need something more complex if that's wanted.
*/
