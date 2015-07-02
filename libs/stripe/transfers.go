package stripe

import (
	"fmt"
	"net/url"
	"strconv"
)

type Transfer struct {
	ID                   string            `json:"id"`
	Object               string            `json:"object"` // "transfer"
	LiveMode             bool              `json:"livemode"`
	Amount               int               `json:"amount"` // cents
	Created              Timestamp         `json:"created"`
	Currency             Currency          `json:"currency"`
	Date                 Timestamp         `json:"date"`
	Status               string            `json:"status"` // paid, pending, failed
	Type                 string            `json:"type"`   // card, bank_account
	BalanceTransaction   string            `json:"balance_transaction"`
	Description          string            `json:"description"`
	Metadata             map[string]string `json:"metadata"`
	RecipientID          string            `json:"recipient"`
	StatementDescription string            `json:"statement_description"`
	BankAccount          *Account          `json:"bank_account"`
	// Card *Card `json:"card"`
}

type CreateTransferRequest struct {
	Amount               int               // required : cents
	Currency             Currency          // required
	RecipientID          string            // required
	Description          string            // optional
	BankAccount          string            // optional
	Card                 string            // optional
	StatementDescription string            // optional
	Metadata             map[string]string // optional
}

func (s *Client) CreateTransfer(req *CreateTransferRequest) (*Transfer, error) {
	params := url.Values{}
	params.Set("amount", strconv.Itoa(req.Amount))
	params.Set("currency", req.Currency.ISO)
	params.Set("recipient", req.RecipientID)
	if req.Description != "description" {
		params.Set("description", req.Description)
	}
	if req.BankAccount != "" {
		params.Set("bank_account", req.BankAccount)
	}
	if req.Card != "" {
		params.Set("card", req.Card)
	}
	if req.StatementDescription != "" {
		params.Set("statement_description", req.StatementDescription)
	}
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			params.Set(fmt.Sprintf("metadata[%s]", k), v)
		}
	}

	var transfer Transfer
	if err := s.query("POST", transfersURL, params, &transfer); err != nil {
		return nil, err
	}
	return &transfer, nil
}

func (s *Client) GetTransfer(id string) (*Transfer, error) {
	var transfer Transfer
	if err := s.query("GET", transfersURL+"/"+id, nil, &transfer); err != nil {
		return nil, err
	}
	return &transfer, nil
}
