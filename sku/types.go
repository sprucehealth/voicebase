package sku

import "fmt"

type SKU string
type SKUCategory string

const (
	AcneVisit     SKU         = "acne_visit"
	AcneFollowup  SKU         = "acne_followup"
	CategoryVisit SKUCategory = "visit"
)

var skuToCategoryMapping = map[SKU]SKUCategory{
	AcneVisit: CategoryVisit,
}

func CategoryForSKU(sku SKU) (SKUCategory, bool) {
	category, ok := skuToCategoryMapping[sku]
	return category, ok
}

func (s SKU) String() string {
	return string(s)
}

func GetSKU(s string) (SKU, error) {
	switch ps := SKU(s); ps {
	case AcneVisit, AcneFollowup:
		return ps, nil
	}

	return SKU(""), fmt.Errorf("%s is not a supported SKU", s)
}

func (s *SKU) Scan(src interface{}) error {

	str, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Cannot scan type %T into SKU when string expected", src)
	}

	var err error
	*s, err = GetSKU(string(str))

	return err
}

func (s *SKU) UnmarshalJSON(data []byte) error {
	strData := string(data)
	var err error
	if len(strData) >= 2 && strData[0] == '"' && strData[len(strData)-1] == '"' {
		*s, err = GetSKU(strData[1 : len(strData)-1])
	} else {
		*s, err = GetSKU(strData)
	}

	return err
}

func (s *SKU) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, s.String())), nil
}
