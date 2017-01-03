package ast

// Directive implements Node
type Directive struct {
	Loc       Location
	Name      *Name
	Arguments []*Argument
}

func (dir *Directive) GetLoc() Location {
	return dir.Loc
}
