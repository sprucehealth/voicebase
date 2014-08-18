package common

import "fmt"

type Currency string

const (
	USD Currency = "USD"
)

func GetCurrency(s string) (Currency, error) {
	switch c := Currency(s); c {
	case USD:
		return c, nil
	}
	return Currency(""), fmt.Errorf("Unknown currency %s", s)
}

func (c Currency) String() string {
	return string(c)
}

func (c *Currency) Scan(src interface{}) error {
	var err error
	switch t := src.(type) {
	case []byte:
		*c, err = GetCurrency(string(t))
	case string:
		*c, err = GetCurrency(t)
	default:
		return fmt.Errorf("common: Cannot scan %T into Currency", src)
	}
	return err
}

type Cost struct {
	Currency string  `json:"currency"`
	Amount   float32 `json:"amount"`
}

type LineItem struct {
	ID          int64  `json:"-"`
	Description string `json:"description"`
	Cost        Cost   `json:"cost"`
	ItemType    string `json:"-"`
}

type CostBreakdown struct {
	LineItems []*LineItem `json:"line_items"`
	TotalCost Cost        `json:"total_cost"`
}

func (c *CostBreakdown) CalculateTotal() {
	var totalCost float32
	var currency string
	// convert into smallest currency unit (for now this is just cents)
	// as it helps with the precision when totalling the line items
	for _, lItem := range c.LineItems {
		currency = lItem.Cost.Currency
		totalCost += lItem.Cost.Amount * 100
	}

	c.TotalCost = Cost{
		Amount:   totalCost / 100.0,
		Currency: currency,
	}
}
