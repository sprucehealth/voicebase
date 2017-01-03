package ast

// Argument implements Node
type Argument struct {
	Loc   Location
	Name  *Name
	Value Value
}

func (arg *Argument) GetLoc() Location {
	return arg.Loc
}
