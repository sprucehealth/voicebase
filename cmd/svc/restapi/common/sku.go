package common

import "fmt"

type SKU struct {
	ID           int64
	Type         string
	CategoryType SKUCategoryType
}

type SKUCategoryType string

func (s SKUCategoryType) String() string {
	return string(s)
}

func NewSKUCategoryType(s string) SKUCategoryType {
	return SKUCategoryType(s)
}

func (s *SKUCategoryType) Scan(src interface{}) error {
	switch sc := src.(type) {
	case string:
		*s = SKUCategoryType(sc)
	case []byte:
		*s = SKUCategoryType(string(sc))
	default:
		return fmt.Errorf("Cannot scan type %T into SKUCategoryType when string expected", src)
	}
	return nil
}

var (
	SCVisit    = SKUCategoryType("visit")
	SCFollowup = SKUCategoryType("followup")
)
