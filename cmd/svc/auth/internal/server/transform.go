package server

import (
	"github.com/sprucehealth/backend/cmd/svc/auth/internal/dal"
	"github.com/sprucehealth/backend/svc/auth"
)

func accountAsResponse(account *dal.Account) *auth.Account {
	return &auth.Account{
		ID:        account.ID.String(),
		FirstName: account.FirstName,
		LastName:  account.LastName,
		Type:      auth.AccountType(auth.AccountType_value[account.Type.String()]),
	}
}
