package ast

// Name implements Node
type Name struct {
	Loc   Location
	Value string
}

func (node *Name) GetLoc() Location {
	return node.Loc
}
