package common

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Currency represents a monetary system
type Currency string

const (
	// USD is a Curreny representing the United States Dollar
	USD          Currency = "USD"
	smallestUnit          = 100.0
)

// ParseCurrency returns the Currency type the maps to the provided string. An error is returned if there is no match.
func ParseCurrency(s string) (Currency, error) {
	switch c := Currency(s); c {
	case USD:
		return c, nil
	}
	return Currency(""), fmt.Errorf("Unknown currency %s", s)
}

func (c Currency) String() string {
	return string(c)
}

// Scan implements the sql.Scanner interface to load Currencies from a data store and provide validation
func (c *Currency) Scan(src interface{}) error {
	var err error
	switch t := src.(type) {
	case []byte:
		*c, err = ParseCurrency(string(t))
	case string:
		*c, err = ParseCurrency(t)
	default:
		return fmt.Errorf("common: Cannot scan %T into Currency", src)
	}
	return err
}

// Cost represents a monetary system and value
type Cost struct {
	Currency string `json:"currency"`
	Amount   int    `json:"amount"`
}

func (c Cost) String() string {
	isNegative := c.Amount < 0
	var marshalledValue []byte
	if isNegative {
		marshalledValue = append(marshalledValue, '-')
	}
	marshalledValue = append(marshalledValue, '$')
	return string(strconv.AppendFloat(marshalledValue, math.Abs(float64(c.Amount))/smallestUnit, 'f', -1, 64))
}

// Charge created the string description associated with the costs "Charge"
func (c Cost) Charge() string {
	isNegative := c.Amount < 0
	var marshalledValue []byte
	if isNegative {
		marshalledValue = append(marshalledValue, '-')
	}
	return string(strconv.AppendFloat(marshalledValue, math.Abs(float64(c.Amount))/smallestUnit, 'f', -1, 64))
}

// ItemCost represents the mapping between item to charge for an the charge itself
// This structure also houses a destailed breakdown of the charges/costs associated with the item
type ItemCost struct {
	ID           int64            `json:"-"`
	SKUType      string           `json:"-"`
	SKUCategory  *SKUCategoryType `json:"-"`
	Status       string           `json:"-"`
	PromoApplied bool             `json:"-"`
	LineItems    []*LineItem      `json:"line_items"`
}

// LineItem represents an individual charge delta to be reported
type LineItem struct {
	ID          int64  `json:"-"`
	Description string `json:"description"`
	Cost        Cost   `json:"cost"`
	SKUType     string `json:"-"`
}

// CostBreakdown represents a collection of ItemCosts and LineItems that make up a TotalCost
type CostBreakdown struct {
	ItemCosts []*ItemCost `json:"-"`
	LineItems []*LineItem `json:"line_items"`
	TotalCost Cost        `json:"total_cost"`
}

func lineItemsTotal(lis []*LineItem) Cost {
	var totalCost int
	var currency string
	for _, li := range lis {
		currency = li.Cost.Currency
		totalCost += li.Cost.Amount
	}
	return Cost{
		Amount:   totalCost,
		Currency: currency,
	}
}

// TotalCost returns a totaled cost gained from the more genualr line item breakdown
func (ic *ItemCost) TotalCost() Cost {
	return lineItemsTotal(ic.LineItems)
}

// CalculateTotal performs an in place operation updating the interal total cost state
func (c *CostBreakdown) CalculateTotal() {
	c.TotalCost = lineItemsTotal(c.LineItems)
	if c.TotalCost.Amount < 0 {
		c.TotalCost.Amount = 0
	}
}

// PatientReceiptStatus is a representation of the state of the charge
type PatientReceiptStatus string

const (
	// PRChargePending is the status of an item when it still needs to be charged
	PRChargePending PatientReceiptStatus = "CHARGE_PENDING"

	// PRCharged is the status of an item when it has been charged
	PRCharged PatientReceiptStatus = "CHARGED"
)

func (p PatientReceiptStatus) String() string {
	return string(p)
}

// Scan implements the sql.Scanner interface to load PatientReceiptStatus from a data store and provide validation
func (p *PatientReceiptStatus) Scan(src interface{}) error {
	var err error
	switch v := src.(type) {
	case []byte:
		*p, err = ParsePatientReceiptStatus(string(v))
	case string:
		*p, err = ParsePatientReceiptStatus(v)
	default:
		return fmt.Errorf("common: Unable to scan %T into PatientReceiptStatus", src)

	}
	return err
}

// ParsePatientReceiptStatus returns the PatientReceiptStatus type the maps to the provided string. An error is returned if there is no match.
func ParsePatientReceiptStatus(s string) (PatientReceiptStatus, error) {
	switch p := PatientReceiptStatus(strings.ToUpper(s)); p {
	case PRChargePending, PRCharged:
		return p, nil
	}
	return PatientReceiptStatus(""), fmt.Errorf("PatientReceiptStatus %s unknown", s)
}

// PatientReceipt represents the patient facing description of their total charge
type PatientReceipt struct {
	ID                int64                `json:"id,string"`
	ReferenceNumber   string               `json:"reference_number"`
	SKUType           string               `json:"item_type"`
	ItemID            int64                `json:"item_id,string"`
	PatientID         int64                `json:"-"`
	StripeChargeID    string               `json:"-"`
	CreationTimestamp time.Time            `json:"creation_timestamp"`
	Status            PatientReceiptStatus `json:"-"`
	ItemCostID        int64                `json:"-"`
	CostBreakdown     *CostBreakdown       `json:"costs"`
}

// DoctorTransaction represents a transaction performed in respect to it's impact on the providing doctor
type DoctorTransaction struct {
	ID         int64
	DoctorID   int64
	ItemCostID *int64
	SKUType    string
	ItemID     int64
	PatientID  int64
	Created    time.Time
}
