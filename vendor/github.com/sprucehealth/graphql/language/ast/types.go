package ast

type Type interface {
	GetLoc() Location
	String() string
}

// Ensure that all value types implements Value interface
var _ Type = (*Named)(nil)
var _ Type = (*List)(nil)
var _ Type = (*NonNull)(nil)

// Named implements Node, Type
type Named struct {
	Loc  Location
	Name *Name
}

func (t *Named) GetLoc() Location {
	return t.Loc
}

func (t *Named) String() string {
	if t.Name != nil {
		return t.Name.Value
	}
	return "Named"
}

// List implements Node, Type
type List struct {
	Loc  Location
	Type Type
}

func (t *List) GetLoc() Location {
	return t.Loc
}

func (t *List) String() string {
	return t.Type.String()
}

// NonNull implements Node, Type
type NonNull struct {
	Loc  Location
	Type Type
}

func (t *NonNull) GetLoc() Location {
	return t.Loc
}

func (t *NonNull) String() string {
	return t.Type.String()
}
