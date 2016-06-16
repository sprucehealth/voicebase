package main

import (
	"fmt"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/language/ast"
	"golang.org/x/net/context"
)

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

func nodeIDPrefix(nodeID string) string {
	i := strings.IndexByte(nodeID, '_')
	if i < 0 {
		return ""
	}
	return nodeID[:i]
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
	g, _ := mei["gender"].(string)

	var dob *directory.Date
	mdob, _ := mei["dob"].(map[string]interface{})
	if mdob != nil {
		dob = &directory.Date{}
		dob.Month = uint32(mdob["month"].(int))
		dob.Day = uint32(mdob["day"].(int))
		dob.Year = uint32(mdob["year"].(int))
	}

	entityInfo := &directory.EntityInfo{
		FirstName:     fn,
		MiddleInitial: mi,
		LastName:      ln,
		GroupName:     gn,
		ShortTitle:    st,
		LongTitle:     lt,
		DOB:           dob,
		Gender:        directory.EntityInfo_Gender(directory.EntityInfo_Gender_value[g]),
		Note:          n,
	}

	return entityInfo, nil
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
		h := r.Header.Get("X-Forwarded-For")
		if ix := strings.IndexByte(h, ','); ix > 0 {
			return h[:ix]
		}
		return h
	}
	return r.RemoteAddr
}

// dedupeStrings returns a slice of strings with duplicates removed. The order is not guaranteed to remain the same.
func dedupeStrings(ss []string) []string {
	if len(ss) == 0 {
		return ss
	}
	mp := make(map[string]struct{}, len(ss))
	for i := 0; i < len(ss); i++ {
		s := ss[i]
		if _, ok := mp[s]; !ok {
			mp[s] = struct{}{}
		} else {
			ss[i] = ss[len(ss)-1]
			ss = ss[:len(ss)-1]
			i--
		}
	}
	return ss
}

func entityInOrgForAccountID(ctx context.Context, ram raccess.ResourceAccessor, orgID string, acc *auth.Account) (*directory.Entity, error) {
	return raccess.EntityInOrgForAccountID(ctx, ram, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}, orgID)
}
