package ast

// Ensure that all typeDefinition types implements Definition interface
var _ Definition = (*ObjectDefinition)(nil)
var _ Definition = (*InterfaceDefinition)(nil)
var _ Definition = (*UnionDefinition)(nil)
var _ Definition = (*ScalarDefinition)(nil)
var _ Definition = (*EnumDefinition)(nil)
var _ Definition = (*InputObjectDefinition)(nil)
var _ Definition = (*TypeExtensionDefinition)(nil)

// ObjectDefinition implements Node, Definition
type ObjectDefinition struct {
	Loc        Location
	Name       *Name
	Interfaces []*Named
	Fields     []*FieldDefinition
}

func (def *ObjectDefinition) GetLoc() Location {
	return def.Loc
}

func (def *ObjectDefinition) GetName() *Name {
	return def.Name
}

func (def *ObjectDefinition) GetVariableDefinitions() []*VariableDefinition {
	return []*VariableDefinition{}
}

func (def *ObjectDefinition) GetSelectionSet() *SelectionSet {
	return &SelectionSet{}
}

func (def *ObjectDefinition) GetOperation() string {
	return ""
}

// FieldDefinition implements Node
type FieldDefinition struct {
	Loc       Location
	Name      *Name
	Arguments []*InputValueDefinition
	Type      Type
}

func (def *FieldDefinition) GetLoc() Location {
	return def.Loc
}

// InputValueDefinition implements Node
type InputValueDefinition struct {
	Loc          Location
	Name         *Name
	Type         Type
	DefaultValue Value
}

func (def *InputValueDefinition) GetLoc() Location {
	return def.Loc
}

// InterfaceDefinition implements Node, Definition
type InterfaceDefinition struct {
	Loc    Location
	Name   *Name
	Fields []*FieldDefinition
}

func (def *InterfaceDefinition) GetLoc() Location {
	return def.Loc
}

func (def *InterfaceDefinition) GetName() *Name {
	return def.Name
}

func (def *InterfaceDefinition) GetVariableDefinitions() []*VariableDefinition {
	return []*VariableDefinition{}
}

func (def *InterfaceDefinition) GetSelectionSet() *SelectionSet {
	return &SelectionSet{}
}

func (def *InterfaceDefinition) GetOperation() string {
	return ""
}

// UnionDefinition implements Node, Definition
type UnionDefinition struct {
	Loc   Location
	Name  *Name
	Types []*Named
}

func (def *UnionDefinition) GetLoc() Location {
	return def.Loc
}

func (def *UnionDefinition) GetName() *Name {
	return def.Name
}

func (def *UnionDefinition) GetVariableDefinitions() []*VariableDefinition {
	return []*VariableDefinition{}
}

func (def *UnionDefinition) GetSelectionSet() *SelectionSet {
	return &SelectionSet{}
}

func (def *UnionDefinition) GetOperation() string {
	return ""
}

// ScalarDefinition implements Node, Definition
type ScalarDefinition struct {
	Loc  Location
	Name *Name
}

func (def *ScalarDefinition) GetLoc() Location {
	return def.Loc
}

func (def *ScalarDefinition) GetName() *Name {
	return def.Name
}

func (def *ScalarDefinition) GetVariableDefinitions() []*VariableDefinition {
	return []*VariableDefinition{}
}

func (def *ScalarDefinition) GetSelectionSet() *SelectionSet {
	return &SelectionSet{}
}

func (def *ScalarDefinition) GetOperation() string {
	return ""
}

// EnumDefinition implements Node, Definition
type EnumDefinition struct {
	Loc    Location
	Name   *Name
	Values []*EnumValueDefinition
}

func (def *EnumDefinition) GetLoc() Location {
	return def.Loc
}

func (def *EnumDefinition) GetName() *Name {
	return def.Name
}

func (def *EnumDefinition) GetVariableDefinitions() []*VariableDefinition {
	return []*VariableDefinition{}
}

func (def *EnumDefinition) GetSelectionSet() *SelectionSet {
	return &SelectionSet{}
}

func (def *EnumDefinition) GetOperation() string {
	return ""
}

// EnumValueDefinition implements Node, Definition
type EnumValueDefinition struct {
	Loc  Location
	Name *Name
}

func (def *EnumValueDefinition) GetLoc() Location {
	return def.Loc
}

// InputObjectDefinition implements Node, Definition
type InputObjectDefinition struct {
	Loc    Location
	Name   *Name
	Fields []*InputValueDefinition
}

func (def *InputObjectDefinition) GetLoc() Location {
	return def.Loc
}

func (def *InputObjectDefinition) GetName() *Name {
	return def.Name
}

func (def *InputObjectDefinition) GetVariableDefinitions() []*VariableDefinition {
	return []*VariableDefinition{}
}

func (def *InputObjectDefinition) GetSelectionSet() *SelectionSet {
	return &SelectionSet{}
}

func (def *InputObjectDefinition) GetOperation() string {
	return ""
}

// TypeExtensionDefinition implements Node, Definition
type TypeExtensionDefinition struct {
	Loc        Location
	Definition *ObjectDefinition
}

func (def *TypeExtensionDefinition) GetLoc() Location {
	return def.Loc
}

func (def *TypeExtensionDefinition) GetVariableDefinitions() []*VariableDefinition {
	return []*VariableDefinition{}
}

func (def *TypeExtensionDefinition) GetSelectionSet() *SelectionSet {
	return &SelectionSet{}
}

func (def *TypeExtensionDefinition) GetOperation() string {
	return ""
}
