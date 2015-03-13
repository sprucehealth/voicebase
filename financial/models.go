package financial

import "time"

// IncomingItem represents an item that we charged for on Spruce.
// eg: patient visit
type IncomingItem struct {

	// ItemID is the ID of the item that was purchased.
	ItemID string

	// SKUType represents a unique SKU of the incoming item.
	SKUType string

	// ReceiptID represents a unique reference to the incoming item.
	ReceiptID string

	// State represents a U.S. state in which the item originated.
	State string

	// ChargeID represents a unique reference to the financial charge that resulted from this item. It's possible
	// for this value to be nil in the event no charge was made for this item.
	ChargeID *string

	// Created reperesents the time at which the transaction was completed.
	Created time.Time
}

// OutgoingItem represents an item that we pay out for on Spruce.
// eg: doctor reviewing a patient case.
type OutgoingItem struct {

	// ItemID is the ID of the outgoing item.
	ItemID string

	// SKUType represents a unique SKU of the item.
	SKUType string

	// ReceiptID represents a unique reference to the receipt of the correspdonding incoming item.
	ReceiptID string

	// State represents a U.S. state in which the corresponding incoming item originated.
	State string

	// Represents the name of the individual that is responsible for this outgoing item.
	Name string

	// Created represents the time at which the outgoing item was completed.
	Created time.Time
}
