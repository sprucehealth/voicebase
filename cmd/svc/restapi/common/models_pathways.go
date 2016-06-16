package common

import "fmt"

// PathwayStatus represents the status of a pathway.
type PathwayStatus string

const (
	// PathwayActive is a pathway that can be used to start a visit.
	PathwayActive PathwayStatus = "ACTIVE"
	// PathwayDeprecated is a pathway that is no longer used and cannot be used to start a visit.
	PathwayDeprecated PathwayStatus = "DEPRECATED"
)

// PathwayMenuItemType is the type of an item in the pathway menu.
type PathwayMenuItemType string

const (
	// PathwayMenuItemTypeMenu is a submenu that looks just like the parent (recursive)
	PathwayMenuItemTypeMenu PathwayMenuItemType = "menu"
	// PathwayMenuItemTypePathway is a pathway which can be chosen to start a visit
	PathwayMenuItemTypePathway PathwayMenuItemType = "pathway"
)

// ParsePathwayStatus validates the provided string to make sure it's a valid pathway status.
func ParsePathwayStatus(s string) (PathwayStatus, error) {
	switch s {
	case string(PathwayActive):
		return PathwayActive, nil
	case string(PathwayDeprecated):
		return PathwayDeprecated, nil
	}
	return "", fmt.Errorf("unknown pathway status '%s'", s)
}

// String implements fmt.Stringer
func (ps PathwayStatus) String() string {
	return string(ps)
}

// Scan implements sql.Scanner. It expects src to be non-nil and of type string or []byte.
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

// String implements fmt.Stringer
func (pt PathwayMenuItemType) String() string {
	return string(pt)
}

type Pathway struct {
	ID             int64           `json:"id,string,omitempty"`
	Tag            string          `json:"tag,omitempty"`
	Name           string          `json:"name,omitempty"`
	MedicineBranch string          `json:"medicine_branch,omitempty"`
	Status         PathwayStatus   `json:"status,omitempty"`
	Details        *PathwayDetails `json:"details,omitempty"`
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
	Menu       *PathwayMenu `json:"menu,omitempty"`
	PathwayTag string       `json:"pathway_tag,omitempty"`
}

// Conditional is used to represent a simple boolean conditional.
//
// (gender == "female") AND (state != "CA") AND (age >= 18)
//   would be represented as
// [Conditional{"==", "gender", "female"}, Conditional{"!=", "state", "CA"}, Conditional{">=", "age", 18}]
//
// No way to do OR conditions or sub-conditions with this design. Need something more complex if that's wanted.
type Conditional struct {
	Op    string      `json:"op"`
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
	Not   bool        `json:"not"`
}

type PathwayDetails struct {
	WhatIsIncluded  []string                 `json:"what_is_included"`
	WhoWillTreatMe  string                   `json:"who_will_treat_me"`
	RightForMe      string                   `json:"right_for_me"`
	DidYouKnow      []string                 `json:"did_you_know"`
	FAQ             []FAQ                    `json:"faq"`
	AgeRestrictions []*PathwayAgeRestriction `json:"age_restrictions,omitempty"`
}

// Validate validate returns false and a reason iff the details are not valid.
func (pd *PathwayDetails) Validate() (bool, string) {
	for _, faq := range pd.FAQ {
		if ok, msg := faq.Validate(); !ok {
			return false, msg
		}
	}
	if len(pd.AgeRestrictions) != 0 {
		lastAge := 0
		for i, ar := range pd.AgeRestrictions {
			// The last element and only the last element should have a null max age
			last := i == len(pd.AgeRestrictions)-1
			if ar.MaxAgeOfRange == nil {
				if !last {
					return false, "Max age range should not be null except for the last value"
				}
			} else {
				if last {
					return false, "Last max age range should be null"
				}
				if *ar.MaxAgeOfRange <= lastAge {
					return false, "Age ranges must be in increasing order and must not overlap"
				}
				lastAge = *ar.MaxAgeOfRange
			}
			if !ar.VisitAllowed && ar.Alert == nil {
				return false, "An alert is required when visit is not allowed"
			}
			if ar.Alert != nil {
				if ok, msg := ar.Alert.Validate(); !ok {
					return false, msg
				}
			}
		}
	}
	return true, ""
}

// FAQ is a frequently asked question
type FAQ struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// Validate validate returns false and a reason iff the faq is not valid.
func (faq *FAQ) Validate() (bool, string) {
	if faq.Question == "" {
		return false, "FAQ question is required"
	}
	if faq.Answer == "" {
		return false, "FAQ answer is required"
	}
	return true, ""
}

// PathwayAgeRestriction is a waterline for a range of ages for a pathway.
type PathwayAgeRestriction struct {
	// MaxAgeOfRange is the age waterline for the current range. It must be null
	// for the last value in the range (and only for the last value).
	MaxAgeOfRange *int `json:"max_age_of_range"`
	// VisitAllowed specifies if the age range is allowed to start a visit. If
	// it is false then Alert must not be nil.
	VisitAllowed bool `json:"visit_allowed"`
	// Alert is used be the app to show an alert modal when the age range is matched.
	// It must not be nil if VisitAllowed is true. However, it's allowed to defined for allowed
	// visits in which case it's advisory (e.g. "must have parent approval").
	Alert *PathwayAlert `json:"alert,omitempty"`
	// AlternatePathwayTag is an optional pathway that will be used for intake if this age range is matched.
	AlternatePathwayTag string `json:"alternate_pathway_tag,omitempty"`
}

// PathwayAlert is the contents of a modal shown to the patient when starting a visit.
type PathwayAlert struct {
	Type        string `json:"type"`
	Title       string `json:"title,omitempty"`
	Message     string `json:"message"`
	ButtonTitle string `json:"button_title"`
}

// Validate validate returns false and a reason iff the alert is not valid.
func (pa *PathwayAlert) Validate() (bool, string) {
	if pa.Type == "" {
		return false, "Alert type required"
	}
	if pa.Message == "" {
		return false, "Alert message required"
	}
	if pa.ButtonTitle == "" {
		return false, "Alert button title required"
	}
	return true, ""
}
