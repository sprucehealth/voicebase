package ast

// Document implements Node
type Document struct {
	Loc         Location
	Definitions []Node
}

func (node *Document) GetLoc() Location {
	return node.Loc
}
