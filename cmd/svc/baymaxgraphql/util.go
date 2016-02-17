package main

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/language/ast"
	"golang.org/x/net/context"
)

type ctxKey int

const (
	ctxAccount ctxKey = iota
	ctxSpruceHeaders
	ctxClientEncryptionKey
	ctxRequestID
	ctxAuthToken
)

type errInvalidContactFormat string

func (e errInvalidContactFormat) Error() string {
	return string(e)
}

func ctxWithSpruceHeaders(ctx context.Context, sh *apiservice.SpruceHeaders) context.Context {
	return context.WithValue(ctx, ctxSpruceHeaders, sh)
}

// spruceHeadersFromContext returns the spruce headers which may be nil
func spruceHeadersFromContext(ctx context.Context) *apiservice.SpruceHeaders {
	sh, _ := ctx.Value(ctxSpruceHeaders).(*apiservice.SpruceHeaders)
	return sh
}

func ctxWithAuthToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, ctxAuthToken, token)
}

func authTokenFromContext(ctx context.Context) string {
	token, _ := ctx.Value(ctxAuthToken).(string)
	return token
}

func ctxWithRequestID(ctx context.Context, id uint64) context.Context {
	return context.WithValue(ctx, ctxRequestID, id)
}

func requestIDFromContext(ctx context.Context) uint64 {
	id, _ := ctx.Value(ctxRequestID).(uint64)
	return id
}

func ctxWithAccount(ctx context.Context, acc *account) context.Context {
	// Never set a nil account so that we can update it in place. It's kind
	// of gross, but can't think of a better way to deal with authenticate
	// needing to update the account at the moment. Ideally the GraphQL pkg would
	// have a way to update context as it went through the executor.. but alas..
	if acc == nil {
		acc = &account{}
	}
	return context.WithValue(ctx, ctxAccount, acc)
}

// accountFromContext returns the account from the context which may be nil
func accountFromContext(ctx context.Context) *account {
	acc, _ := ctx.Value(ctxAccount).(*account)
	if acc != nil && acc.ID == "" {
		return nil
	}
	return acc
}

func ctxWithClientEncryptionKey(ctx context.Context, clientEncryptionKey string) context.Context {
	// The client encryption key is generated at token validation check time, so we store it here to make it available to concerned parties
	return context.WithValue(ctx, ctxClientEncryptionKey, clientEncryptionKey)
}

// clientEncryptionKeyFromContext returns the clientEncryptionKey from the context which may be the empty string
func clientEncryptionKeyFromContext(ctx context.Context) string {
	cek, _ := ctx.Value(ctxClientEncryptionKey).(string)
	return cek
}

func serviceFromParams(p graphql.ResolveParams) *service {
	return p.Info.RootValue.(map[string]interface{})["service"].(*service)
}

func selectedFields(p graphql.ResolveParams) []string {
	f := p.Info.FieldASTs[0]
	fields := make([]string, 0, len(f.SelectionSet.Selections))
	for _, s := range f.SelectionSet.Selections {
		if f, ok := s.(*ast.Field); ok && f.Name != nil {
			fields = append(fields, f.Name.Value)
		}
	}
	return fields
}

func selectingOnlyID(p graphql.ResolveParams) bool {
	f := p.Info.FieldASTs[0]
	if len(f.SelectionSet.Selections) > 1 {
		return false
	}
	for _, s := range f.SelectionSet.Selections {
		if f, ok := s.(*ast.Field); ok && f.Name != nil {
			if f.Name.Value == "id" {
				return true
			}
		}
	}
	return false
}

func nodePrefix(nodeID string) string {
	i := strings.IndexByte(nodeID, '_')
	prefix := nodeID[:i]

	return prefix
}

func contactFromInput(input interface{}) (*directory.Contact, error) {
	mci, ok := input.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Unable to parse input contact data: %+v", input)
	}

	id, _ := mci["id"].(string)
	t, _ := mci["type"].(string)
	v, _ := mci["value"].(string)
	l, _ := mci["label"].(string)

	ct, ok := directory.ContactType_value[t]
	if !ok {
		return nil, fmt.Errorf("Unknown contact type: %q", t)
	}

	var formattedValue string
	var err error
	switch directory.ContactType(ct) {
	case directory.ContactType_PHONE:
		formattedValue, err = phone.Format(v, phone.E164)
		if err != nil {
			return nil, errInvalidContactFormat(err.Error())
		}
	case directory.ContactType_EMAIL:
		if !validate.Email(v) {
			return nil, errInvalidContactFormat("Invalid email " + v)
		}
		formattedValue = v
	}

	return &directory.Contact{
		ID:          id,
		Value:       formattedValue,
		ContactType: directory.ContactType(ct),
		Label:       l,
	}, nil
}

func contactListFromInput(input []interface{}, ignoreInvalid bool) ([]*directory.Contact, error) {
	contacts := make([]*directory.Contact, 0, len(input))
	for _, ci := range input {
		c, err := contactFromInput(ci)
		if _, ok := errors.Cause(err).(errInvalidContactFormat); ok && ignoreInvalid {
			continue
		} else if err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}
	return contacts, nil
}

func entityInfoFromInput(ei interface{}) (*directory.EntityInfo, error) {
	mei, ok := ei.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Unable to parse input entity info data: %+v", ei)
	}

	fn, _ := mei["firstName"].(string)
	mi, _ := mei["middleInitial"].(string)
	ln, _ := mei["lastName"].(string)
	gn, _ := mei["groupName"].(string)
	st, _ := mei["shortTitle"].(string)
	lt, _ := mei["longTitle"].(string)
	n, _ := mei["note"].(string)

	// If no display name was provided then build one from our input

	entityInfo := &directory.EntityInfo{
		FirstName:     fn,
		MiddleInitial: mi,
		LastName:      ln,
		GroupName:     gn,
		ShortTitle:    st,
		LongTitle:     lt,
		Note:          n,
	}

	return entityInfo, nil
}

func buildDisplayName(info *directory.EntityInfo, contacts []*directory.Contact) (string, error) {
	if info.FirstName != "" || info.LastName != "" {
		var displayName string
		if info.MiddleInitial != "" {
			displayName = info.FirstName + " " + info.MiddleInitial + ". " + info.LastName
		} else {
			displayName = info.FirstName + " " + info.LastName
		}
		if info.ShortTitle != "" {
			displayName += ", " + info.ShortTitle
		}
		return displayName, nil
	} else if info.GroupName != "" {
		return info.GroupName, nil
	}

	// pick the display name to be the first contact value
	for _, c := range contacts {
		if c.ContactType == directory.ContactType_PHONE {
			pn, err := phone.Format(c.Value, phone.Pretty)
			if err != nil {
				return c.Value, nil
			}
			return pn, nil
		}
		return c.Value, nil
	}

	return "", errors.New("Display name cannot be empty and not enough information was supplied to infer one")
}

// isValidPlane0Unicode returns true iff the provided string only has valid plane 0 unicode (no emoji)
func isValidPlane0Unicode(s string) bool {
	for _, r := range s {
		if !utf8.ValidRune(r) {
			return false
		}
		if utf8.RuneLen(r) > 3 {
			return false
		}
	}
	return true
}
