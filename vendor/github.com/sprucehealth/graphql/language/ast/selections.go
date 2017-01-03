package ast

type Selection interface {
	Node
}

// Ensure that all definition types implements Selection interface
var _ Selection = (*Field)(nil)
var _ Selection = (*FragmentSpread)(nil)
var _ Selection = (*InlineFragment)(nil)

// Field implements Node, Selection
type Field struct {
	Loc          Location
	Alias        *Name
	Name         *Name
	Arguments    []*Argument
	Directives   []*Directive
	SelectionSet *SelectionSet
}

func (f *Field) GetLoc() Location {
	return f.Loc
}

// FragmentSpread implements Node, Selection
type FragmentSpread struct {
	Loc        Location
	Name       *Name
	Directives []*Directive
}

func (fs *FragmentSpread) GetLoc() Location {
	return fs.Loc
}

// InlineFragment implements Node, Selection
type InlineFragment struct {
	Loc           Location
	TypeCondition *Named
	Directives    []*Directive
	SelectionSet  *SelectionSet
}

func (f *InlineFragment) GetLoc() Location {
	return f.Loc
}

// SelectionSet implements Node
type SelectionSet struct {
	Loc        Location
	Selections []Selection
}

func (ss *SelectionSet) GetLoc() Location {
	return ss.Loc
}
