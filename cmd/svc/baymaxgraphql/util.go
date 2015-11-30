package main

import (
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

// type GlobalID struct {
// 	Type string `json:"type"`
// 	ID   string `json:"id"`
// }

// // ToGlobalID maps a type and a local ID into a global identifier
// func ToGlobalID(ttype, id string) string {
// 	return strings.TrimRight(base64.StdEncoding.EncodeToString([]byte(ttype+":"+id)), "=")
// }

// // FromGlobalID parses a global identifier to a type and local ID
// func FromGlobalID(globalID string) *GlobalID {
// 	b, err := base64.StdEncoding.DecodeString(globalID)
// 	if err != nil {
// 		return nil
// 	}
// 	strID := string(b)
// 	i := strings.IndexByte(strID, ':')
// 	if i < 0 {
// 		return nil
// 	}
// 	return &GlobalID{
// 		Type: strID[:i],
// 		ID:   strID[i+1:],
// 	}
// }

// func GlobalIDField(typeName string) *graphql.Field {
// 	return &graphql.Field{
// 		Name:        "id",
// 		Description: "The ID of an object",
// 		Type:        graphql.NewNonNull(graphql.ID),
// 		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
// 			v := reflect.ValueOf(p.Source)
// 			if v.Kind() == reflect.Ptr {
// 				v = v.Elem()
// 			}
// 			f := v.FieldByName("ID")
// 			return ToGlobalID(typeName, f.String()), nil
// 		},
// 	}
// }

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
