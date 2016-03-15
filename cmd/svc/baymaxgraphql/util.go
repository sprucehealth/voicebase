package main

import (
	"fmt"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/language/ast"
	"google.golang.org/grpc"
)

// go vet doesn't like that the first argument to grpcErrorf is not a string so alias the function with a different name :(
var grpcErrorf = grpc.Errorf

type errInvalidContactFormat string

func (e errInvalidContactFormat) Error() string {
	return string(e)
}

func serviceFromParams(p graphql.ResolveParams) *service {
	return p.Info.RootValue.(map[string]interface{})["service"].(*service)
}

func remoteAddrFromParams(p graphql.ResolveParams) string {
	return p.Info.RootValue.(map[string]interface{})["remoteAddr"].(string)
}

func userAgentFromParams(p graphql.ResolveParams) string {
	return p.Info.RootValue.(map[string]interface{})["userAgent"].(string)
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

func initialsForEntity(e *models.Entity) string {
	var first, last rune
	if e.FirstName != "" {
		first, _ = utf8.DecodeRuneInString(e.FirstName)
		first = unicode.ToUpper(first)
	}
	if e.LastName != "" {
		last, _ = utf8.DecodeRuneInString(e.LastName)
		last = unicode.ToUpper(last)
	}
	if first == 0 {
		if last == 0 {
			return ""
		}
		return string(last)
	}
	if last == 0 {
		return string(first)
	}
	return string(first) + string(last)
}

func remoteAddrFromRequest(r *http.Request, behindProxy bool) string {
	if behindProxy {
		addrs := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
		return addrs[0]
	}
	return r.RemoteAddr
}
