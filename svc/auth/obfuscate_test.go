package auth

import (
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestObfuscateAccount(t *testing.T) {
	cases := map[string]struct {
		Acc      *Account
		Expected *Account
	}{
		"FullyObfuscatedPatient": {
			Acc: &Account{
				ID:        "account_id",
				FirstName: "First",
				LastName:  "Last",
				Type:      AccountType_PATIENT,
			},
			Expected: &Account{
				ID:        "account_id",
				FirstName: "F",
				LastName:  "L",
				Type:      AccountType_PATIENT,
			},
		},
		"FullyObfuscatedProvider": {
			Acc: &Account{
				ID:        "account_id",
				FirstName: "First",
				LastName:  "Last",
				Type:      AccountType_PROVIDER,
			},
			Expected: &Account{
				ID:        "account_id",
				FirstName: "F",
				LastName:  "L",
				Type:      AccountType_PROVIDER,
			},
		},
		"ObfuscateEmpty": {
			Acc: &Account{
				ID:        "account_id",
				FirstName: "",
				LastName:  "",
				Type:      AccountType_PATIENT,
			},
			Expected: &Account{
				ID:        "account_id",
				FirstName: "",
				LastName:  "",
				Type:      AccountType_PATIENT,
			},
		},
		"ObfuscateSingle": {
			Acc: &Account{
				ID:        "account_id",
				FirstName: "1",
				LastName:  "L",
				Type:      AccountType_PATIENT,
			},
			Expected: &Account{
				ID:        "account_id",
				FirstName: "1",
				LastName:  "L",
				Type:      AccountType_PATIENT,
			},
		},
	}
	for cn, c := range cases {
		test.EqualsCase(t, cn, c.Expected, ObfuscateAccount(c.Acc))
	}
}
