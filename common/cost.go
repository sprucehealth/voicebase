package common

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"
)

type Currency string

const (
	USD          Currency = "USD"
	smallestUnit          = 100.0
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
	Currency string `json:"currency"`
	Amount   int    `json:"amount"`
}

func (c *Cost) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"currency": c.Currency,
		"amount":   float32(c.Amount) / smallestUnit,
	})
}

func (c *Cost) String() string {
	isNegative := c.Amount < 0
	var marshalledValue []byte
	if isNegative {
		marshalledValue = append(marshalledValue, '-')
	}
	marshalledValue = append(marshalledValue, '$')
	return string(strconv.AppendFloat(marshalledValue, math.Abs(float64(c.Amount))/smallestUnit, 'f', -1, 64))
}

type ItemCost struct {
	ID        int64       `json:"-"`
	ItemType  string      `json:"-"`
	Status    string      `json:"-"`
	LineItems []*LineItem `json:"line_items"`
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
	var totalCost int
	var currency string

	for _, lItem := range c.LineItems {
		currency = lItem.Cost.Currency
		totalCost += lItem.Cost.Amount
	}

	c.TotalCost = Cost{
		Amount:   totalCost,
		Currency: currency,
	}
}

type PatientReceiptStatus string

const (
	PRChargePending PatientReceiptStatus = "CHARGE_PENDING"
	PREmailPending  PatientReceiptStatus = "EMAIL_PENDING"
	PREmailSent     PatientReceiptStatus = "EMAIL_SENT"
)

func (p PatientReceiptStatus) String() string {
	return string(p)
}

func (p *PatientReceiptStatus) Scan(src interface{}) error {
	var err error
	switch v := src.(type) {
	case []byte:
		*p, err = GetPatientReceiptStatus(string(v))
	case string:
		*p, err = GetPatientReceiptStatus(v)
	default:
		return fmt.Errorf("common: Unable to scan %T into PatientReceiptStatus", src)

	}
	return err
}

func GetPatientReceiptStatus(s string) (PatientReceiptStatus, error) {
	switch p := PatientReceiptStatus(s); p {
	case PRChargePending, PREmailPending, PREmailSent:
		return p, nil
	}
	return PatientReceiptStatus(""), fmt.Errorf("PatientReceiptStatus %s unknown", s)
}

type PatientReceipt struct {
	ID                int64                `json:"id,string"`
	ReferenceNumber   string               `json:"reference_number"`
	ItemType          string               `json:"item_type"`
	ItemID            int64                `json:"item_id,string"`
	PatientID         int64                `json:"-"`
	CreditCardID      int64                `json:"-"`
	StripeChargeID    string               `json:"-"`
	CreationTimestamp time.Time            `json:"creation_timestamp"`
	Status            PatientReceiptStatus `json:"-"`
	CostBreakdown     *CostBreakdown       `json:"costs"`
}
