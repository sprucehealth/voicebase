package models

import "github.com/sprucehealth/backend/svc/auth"

// Account represents the various aspects of an account in the baymax system
type Account struct {
	ID          string `json:"id"`
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
}

// TransformAccountToModel transforms the auth service account to a graphql representation
func TransformAccountToModel(a *auth.Account, email, phoneNumber string) *Account {
	account := &Account{
		ID:          a.ID,
		FirstName:   a.FirstName,
		LastName:    a.LastName,
		Type:        a.Type.String(),
		Status:      a.Status,
		Email:       email,
		PhoneNumber: phoneNumber,
	}
	// For now do not expose the names and info for patients until we have auditing and PHI exposure tracking
	if a.Type != auth.AccountType_PROVIDER {
		account.FirstName = a.Type.String()
		account.LastName = a.Type.String()
		account.Email = a.Type.String()
		account.PhoneNumber = a.Type.String()
	}
	return account
}
