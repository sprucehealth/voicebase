package gqlintrospect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
)

// Query is the query made to introspect the GraphQL schema
const Query = `
	fragment basicType on __Type {
		kind
		name
		description
		ofType {
			kind
			name
			description
			ofType {
				kind
				name
				description
				ofType {
					kind
					name
					description
					ofType {
						kind
						name
						description
					}
				}
			}
		}
	}

	fragment inputValue on __InputValue {
		name
		description
		type {
			...basicType
		}
	}

	fragment field on __Field {
		name
		description
		type {
			...basicType
		}
		args {
			...inputValue
		}
		isDeprecated
		deprecationReason
	}

	query _ {
		__schema {
			types {
				kind
				name
				description
				fields(includeDeprecated: true) {
					...field
				}
				interfaces {
					...basicType
				}
				possibleTypes {
					...basicType
				}
				enumValues(includeDeprecated: true) {
					name
					description
					isDeprecated
					deprecationReason
				}
				inputFields {
					...inputValue
				}
				ofType {
					...basicType
				}
			}
			queryType {
				name
			}
			mutationType {
				name
			}
		}
	}
`

type request struct {
	Query     string      `json:"query"`
	Variables interface{} `json:"variables,omitempty"`
}

type response struct {
	Data   interface{} `json:"data"`
	Errors Errors      `json:"errors"`
}

type Errors []*Error

type Error struct {
	Message   string           `json:"message"`
	Locations []*ErrorLocation `json:"locations"`
}

type ErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

func (e Errors) Error() string {
	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return fmt.Sprintf("gqlintrospect: %+v", ([]*Error)(e))
	}
	return fmt.Sprintf("gqlintrospect: %s", string(b))
}

// QuerySchema queries a remote GraphQL api for its schema
func QuerySchema(url string) (*Schema, error) {
	req := &request{
		Query: Query,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	hres, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer hres.Body.Close()
	if hres.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(hres.Body)
		if len(b) > 256 {
			b = b[:256]
		}
		return nil, fmt.Errorf("gqlintrospect: expected 200 got %d: %s", hres.StatusCode, string(b))
	}

	schema := &Schema{}
	res := &response{
		Data: &struct {
			Schema *Schema `json:"__schema"`
		}{
			Schema: schema,
		},
	}
	if err := json.NewDecoder(hres.Body).Decode(&res); err != nil {
		return nil, err
	}

	if len(res.Errors) != 0 {
		return nil, res.Errors
	}

	return schema, nil
}

// Fdump prints out the schema to the provided io.Writer
func (schema *Schema) Fdump(w io.Writer) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("runtime error: %+v", r)
			}
		}
	}()

	queryName := *schema.QueryType.Name
	mutationName := ""
	if schema.MutationType != nil {
		mutationName = *schema.MutationType.Name
	}

	sort.Sort(typesName(schema.Types))

	// Enums
	for _, t := range schema.Types {
		// Ignore introspection types
		if t.Name == nil || strings.HasPrefix(*t.Name, "__") {
			continue
		}
		if t.Kind != Enum {
			continue
		}
		if err := printEnum(w, t); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	// Interfaces
	for _, t := range schema.Types {
		// Ignore introspection types
		if t.Name == nil || strings.HasPrefix(*t.Name, "__") {
			continue
		}
		if t.Kind != Interface {
			continue
		}
		if err := printInterface(w, t); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	// Objects (except query and mutation)
	for _, t := range schema.Types {
		// Ignore introspection types
		if t.Name == nil || strings.HasPrefix(*t.Name, "__") {
			continue
		}
		if t.Kind != Object {
			continue
		}
		if *t.Name == queryName || *t.Name == mutationName {
			continue
		}
		if err := printObject(w, t); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	// Unions
	for _, t := range schema.Types {
		// Ignore introspection types
		if t.Name == nil || strings.HasPrefix(*t.Name, "__") {
			continue
		}
		if t.Kind != Union {
			continue
		}
		if err := printUnion(w, t); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	// Input objects
	for _, t := range schema.Types {
		// Ignore introspection types
		if t.Name == nil || strings.HasPrefix(*t.Name, "__") {
			continue
		}
		if t.Kind != InputObject {
			continue
		}
		if err := printInputObject(w, t); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	// Query object
	for _, t := range schema.Types {
		if t.Name == nil || *t.Name != queryName {
			continue
		}
		if err := printObject(w, t); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	// Muration object
	for _, t := range schema.Types {
		if t.Name == nil || *t.Name != mutationName {
			continue
		}
		if err := printObject(w, t); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	return nil
}
