package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

var (
	flagURL = flag.String("url", "", "URL of GraphQL endpoint")
)

type request struct {
	Query     string      `json:"query"`
	Variables interface{} `json:"variables,omitempty"`
}

type response struct {
	Data   interface{}      `json:"data"`
	Errors []*responseError `json:"errors"`
}

type responseError struct {
	Message   string           `json:"message"`
	Locations []*errorLocation `json:"locations"`
}

type errorLocation struct {
	Line   int `json:"Line"`
	Column int `json:"Column"`
}

func main() {
	flag.Parse()
	if *flagURL == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	query := `
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

	req := &request{
		Query: query,
	}
	body, err := json.Marshal(req)
	if err != nil {
		log.Fatal(err)
	}
	hres, err := http.Post(*flagURL, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	defer hres.Body.Close()
	if hres.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(hres.Body)
		log.Fatalf("Expected 200 got %d: %s", hres.StatusCode, string(b))
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
		log.Fatal(err)
	}

	if len(res.Errors) != 0 {
		b, err := json.MarshalIndent(res.Errors, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(os.Stderr, "Errors:\n%s\n", string(b))
		os.Exit(2)
	}

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
		printEnum(t)
		fmt.Println()
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
		printInterface(t)
		fmt.Println()
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
		printObject(t)
		fmt.Println()
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
		printUnion(t)
		fmt.Println()
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
		printInputObject(t)
		fmt.Println()
	}

	// Query object
	for _, t := range schema.Types {
		if t.Name == nil || *t.Name != queryName {
			continue
		}
		printObject(t)
		fmt.Println()
	}

	// Muration object
	for _, t := range schema.Types {
		if t.Name == nil || *t.Name != mutationName {
			continue
		}
		printObject(t)
		fmt.Println()
	}
}

func printUnion(t *Type) {
	if t.Description != nil {
		fmt.Printf("// %s\n", *t.Description)
	}
	sort.Sort(typesName(t.PossibleTypes))
	fmt.Printf("union %s = ", *t.Name)
	for i, t := range t.PossibleTypes {
		if i != 0 {
			fmt.Print(" | ")
		}
		fmt.Print(typeName(t))
	}
	fmt.Println()
}

func printEnum(t *Type) {
	sort.Sort(enumsName(t.EnumValues))
	if t.Description != nil {
		fmt.Printf("// %s\n", *t.Description)
	}
	fmt.Printf("enum %s {\n", *t.Name)
	for _, e := range t.EnumValues {
		if e.IsDeprecated {
			printDeprecation(e.DeprecationReason)
		}
		fmt.Print("\t", e.Name)
		if e.Description != nil {
			fmt.Printf("\t// %s\n", *e.Description)
		} else {
			fmt.Println()
		}
	}
	fmt.Println("}")
}

func printObject(t *Type) {
	if t.Description != nil {
		fmt.Printf("// %s\n", *t.Description)
	}
	sort.Sort(typesName(t.Interfaces))
	fmt.Printf("type %s", *t.Name)
	for _, in := range t.Interfaces {
		fmt.Printf(" : %s", *in.Name)
	}
	sort.Sort(fieldsName(t.Fields))
	fmt.Println(" {")
	for _, f := range t.Fields {
		if f.IsDeprecated {
			printDeprecation(f.DeprecationReason)
		}
		fmt.Printf("\t%s", f.Name)
		printArgs(f.Args)
		fmt.Printf(": %s", typeName(f.Type))
		fmt.Println()
	}
	fmt.Println("}")
}

func printInputObject(t *Type) {
	if t.Description != nil {
		fmt.Printf("// %s\n", *t.Description)
	}
	sort.Sort(inputValuesName(t.InputFields))
	fmt.Printf("input %s {\n", *t.Name)
	for _, f := range t.InputFields {
		fmt.Printf("\t%s: %s", f.Name, typeName(f.Type))
		if f.DefaultValue != nil {
			fmt.Printf(" = %s", strconv.Quote(*f.DefaultValue))
		}
		if f.Description != nil {
			fmt.Printf("\t// %s", *f.Description)
		}
		fmt.Println()
	}
	fmt.Println("}")
}

func printInterface(t *Type) {
	if t.Description != nil {
		fmt.Printf("// %s\n", *t.Description)
	}
	if len(t.PossibleTypes) != 0 {
		fmt.Printf("// Implemented by types:")
		for _, pt := range t.PossibleTypes {
			fmt.Printf(" %s", *pt.Name)
		}
		fmt.Println()
	}
	sort.Sort(fieldsName(t.Fields))
	fmt.Printf("interface %s {\n", *t.Name)
	for _, f := range t.Fields {
		if f.IsDeprecated {
			printDeprecation(f.DeprecationReason)
		}
		fmt.Printf("\t%s", f.Name)
		printArgs(f.Args)
		fmt.Printf(": %s", typeName(f.Type))
		fmt.Println()
	}
	fmt.Println("}")
}

func printDeprecation(reason *string) {
	if reason != nil {
		fmt.Printf("\t// DEPRECATED: %s\n", *reason)
	} else {
		fmt.Println("\t// DEPRECATED")
	}
}

func printArgs(args []*InputValue) {
	if len(args) == 0 {
		return
	}
	sort.Sort(inputValuesName(args))
	as := make([]string, len(args))
	for i, a := range args {
		as[i] = fmt.Sprintf("%s: %s", a.Name, typeName(a.Type))
		if a.DefaultValue != nil {
			as[i] += fmt.Sprintf(" = %s", strconv.Quote(*a.DefaultValue))
		}
		if a.Description != nil {
			as[i] += fmt.Sprintf(" /* %s */", *a.Description)
		}
	}
	fmt.Printf("(%s)", strings.Join(as, ", "))
}

func typeName(t *Type) string {
	if t.Name != nil {
		return *t.Name
	}
	switch t.Kind {
	case NonNull:
		return typeName(t.OfType) + "!"
	case List:
		return "[" + typeName(t.OfType) + "]"
	}
	log.Fatalf("Unable to resolve name of type %+v", t)
	return ""
}

type typesName []*Type

func (n typesName) Len() int           { return len(n) }
func (n typesName) Swap(a, b int)      { n[a], n[b] = n[b], n[a] }
func (n typesName) Less(a, b int) bool { return lessPtrStrLess(n[a].Name, n[b].Name) }

type fieldsName []*Field

func (n fieldsName) Len() int           { return len(n) }
func (n fieldsName) Swap(a, b int)      { n[a], n[b] = n[b], n[a] }
func (n fieldsName) Less(a, b int) bool { return n[a].Name < n[b].Name }

type enumsName []*EnumValue

func (n enumsName) Len() int           { return len(n) }
func (n enumsName) Swap(a, b int)      { n[a], n[b] = n[b], n[a] }
func (n enumsName) Less(a, b int) bool { return n[a].Name < n[b].Name }

type inputValuesName []*InputValue

func (n inputValuesName) Len() int           { return len(n) }
func (n inputValuesName) Swap(a, b int)      { n[a], n[b] = n[b], n[a] }
func (n inputValuesName) Less(a, b int) bool { return n[a].Name < n[b].Name }

func lessPtrStrLess(a, b *string) bool {
	if b == nil {
		return true
	}
	if a == nil {
		return false
	}
	return *a < *b
}

type Schema struct {
	Types        []*Type      `json:"types"`
	QueryType    *Type        `json:"queryType"`
	MutationType *Type        `json:"mutationType"`
	Directives   []*Directive `json:"directives"`
}

type Type struct {
	Kind        TypeKind `json:"kind"`
	Name        *string  `json:"name"`
	Description *string  `json:"description"`

	// OBJECT and INTERFACE only
	Fields []*Field `json:"fields"`

	// OBJECT only
	Interfaces []*Type `json:"interfaces"`

	// INTERFACE and UNION only
	PossibleTypes []*Type `json:"possibleTypes"`

	// ENUM only
	EnumValues []*EnumValue `json:"enumValues"`

	// INPUT_OBJECT only
	InputFields []*InputValue `json:"inputFields"`

	// NON_NULL and LIST only
	OfType *Type `json:"ofType"`
}

type Field struct {
	Name              string        `json:"name"`
	Description       *string       `json:"description"`
	Args              []*InputValue `json:"args"`
	Type              *Type         `json:"type"`
	IsDeprecated      bool          `json:"isDeprecated"`
	DeprecationReason *string       `json:"deprecationReason"`
}

type InputValue struct {
	Name         string  `json:"name"`
	Description  *string `json:"description"`
	Type         *Type   `json:"type"`
	DefaultValue *string `json:"defaultValue"`
}

type EnumValue struct {
	Name              string  `json:"name"`
	Description       *string `json:"description"`
	IsDeprecated      bool    `json:"isDeprecated"`
	DeprecationReason *string `json:"deprecationReason"`
}

type TypeKind string

const (
	Scalar      TypeKind = "SCALAR"
	Object      TypeKind = "OBJECT"
	Interface   TypeKind = "INTERFACE"
	Union       TypeKind = "UNION"
	Enum        TypeKind = "ENUM"
	InputObject TypeKind = "INPUT_OBJECT"
	List        TypeKind = "LIST"
	NonNull     TypeKind = "NON_NULL"
)

type Directive struct {
	Name        string        `json:"name"`
	Description *string       `json:"description"`
	Args        []*InputValue `json:"args"`
	OnOperation bool          `json:"onOperation"`
	OnFragment  bool          `json:"onFragment"`
	OnField     bool          `json:"onField"`
}
