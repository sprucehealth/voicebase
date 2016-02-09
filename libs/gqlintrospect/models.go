package gqlintrospect

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
