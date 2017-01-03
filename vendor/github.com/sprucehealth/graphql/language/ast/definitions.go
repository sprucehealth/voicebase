package ast

type Definition interface {
	// TODO: determine the minimal set of interface for `Definition`
	GetOperation() string
	GetVariableDefinitions() []*VariableDefinition
	GetSelectionSet() *SelectionSet
}

// Ensure that all definition types implements Definition interface
var _ Definition = (*OperationDefinition)(nil)
var _ Definition = (*FragmentDefinition)(nil)
var _ Definition = (Definition)(nil)

// OperationDefinition implements Node, Definition
type OperationDefinition struct {
	Loc                 Location
	Operation           string
	Name                *Name
	VariableDefinitions []*VariableDefinition
	Directives          []*Directive
	SelectionSet        *SelectionSet
}

func (op *OperationDefinition) GetLoc() Location {
	return op.Loc
}

func (op *OperationDefinition) GetOperation() string {
	return op.Operation
}

func (op *OperationDefinition) GetName() *Name {
	return op.Name
}

func (op *OperationDefinition) GetVariableDefinitions() []*VariableDefinition {
	return op.VariableDefinitions
}

func (op *OperationDefinition) GetDirectives() []*Directive {
	return op.Directives
}

func (op *OperationDefinition) GetSelectionSet() *SelectionSet {
	return op.SelectionSet
}

// FragmentDefinition implements Node, Definition
type FragmentDefinition struct {
	Loc                 Location
	Operation           string
	Name                *Name
	VariableDefinitions []*VariableDefinition
	TypeCondition       *Named
	Directives          []*Directive
	SelectionSet        *SelectionSet
}

func (fd *FragmentDefinition) GetLoc() Location {
	return fd.Loc
}

func (fd *FragmentDefinition) GetOperation() string {
	return fd.Operation
}

func (fd *FragmentDefinition) GetName() *Name {
	return fd.Name
}

func (fd *FragmentDefinition) GetVariableDefinitions() []*VariableDefinition {
	return fd.VariableDefinitions
}

func (fd *FragmentDefinition) GetSelectionSet() *SelectionSet {
	return fd.SelectionSet
}

// VariableDefinition implements Node
type VariableDefinition struct {
	Loc          Location
	Variable     *Variable
	Type         Type
	DefaultValue Value
}

func (vd *VariableDefinition) GetLoc() Location {
	return vd.Loc
}
