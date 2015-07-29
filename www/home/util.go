package home

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/www"
)

type statesByAbbr []*common.State

func (s statesByAbbr) Len() int           { return len(s) }
func (s statesByAbbr) Less(a, b int) bool { return s[a].Abbreviation < s[b].Abbreviation }
func (s statesByAbbr) Swap(a, b int)      { s[a], s[b] = s[b], s[a] }

type statesByName []*common.State

func (s statesByName) Len() int           { return len(s) }
func (s statesByName) Less(a, b int) bool { return s[a].Name < s[b].Name }
func (s statesByName) Swap(a, b int)      { s[a], s[b] = s[b], s[a] }

func newParentalConsentCookie(childPatientID int64, token string, r *http.Request) *http.Cookie {
	return www.NewCookie(fmt.Sprintf("ct_%d", childPatientID), token, r)
}

func parentalConsentCookie(childPatientID int64, r *http.Request) string {
	cookie, err := r.Cookie(fmt.Sprintf("ct_%d", childPatientID))
	if err != nil {
		return ""
	}
	return cookie.Value
}
