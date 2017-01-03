package parser

import (
	"fmt"

	"github.com/sprucehealth/graphql/gqlerrors"
	"github.com/sprucehealth/graphql/language/ast"
	"github.com/sprucehealth/graphql/language/lexer"
	"github.com/sprucehealth/graphql/language/source"
)

type parseFn func(parser *Parser) (interface{}, error)

type ParseOptions struct {
	NoSource bool
}

type ParseParams struct {
	Source  interface{}
	Options ParseOptions
}

type Parser struct {
	Lexer   *lexer.Lexer
	Source  *source.Source
	Options ParseOptions
	PrevEnd int
	Token   lexer.Token
}

func Parse(p ParseParams) (*ast.Document, error) {
	var sourceObj *source.Source
	switch p.Source.(type) {
	case *source.Source:
		sourceObj = p.Source.(*source.Source)
	default:
		body, _ := p.Source.(string)
		sourceObj = source.New("GraphQL", body)
	}
	parser, err := makeParser(sourceObj, p.Options)
	if err != nil {
		return nil, err
	}
	doc, err := parseDocument(parser)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

// Converts a name lex token into a name parse node.
func parseName(parser *Parser) (*ast.Name, error) {
	token, err := expect(parser, lexer.NAME)
	if err != nil {
		return nil, err
	}
	return &ast.Name{
		Value: token.Value,
		Loc:   loc(parser, token.Start),
	}, nil
}

func makeParser(s *source.Source, opts ParseOptions) (*Parser, error) {
	p := &Parser{
		Lexer:   lexer.New(s),
		Source:  s,
		Options: opts,
		PrevEnd: 0,
	}
	var err error
	p.Token, err = nextToken(p)
	return p, err
}

/* Implements the parsing rules in the Document section. */

func parseDocument(parser *Parser) (*ast.Document, error) {
	start := parser.Token.Start
	var nodes []ast.Node
	for {
		if skp, err := skip(parser, lexer.EOF); err != nil {
			return nil, err
		} else if skp {
			break
		}
		switch {
		case peek(parser, lexer.BRACE_L):
			node, err := parseOperationDefinition(parser)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		case peek(parser, lexer.NAME):
			switch parser.Token.Value {
			case "query", "mutation", "subscription": // Note: subscription is an experimental non-spec addition.
				node, err := parseOperationDefinition(parser)
				if err != nil {
					return nil, err
				}
				nodes = append(nodes, node)
			case "fragment":
				node, err := parseFragmentDefinition(parser)
				if err != nil {
					return nil, err
				}
				nodes = append(nodes, node)
			case "type":
				node, err := parseObjectTypeDefinition(parser)
				if err != nil {
					return nil, err
				}
				nodes = append(nodes, node)
			case "interface":
				node, err := parseInterfaceTypeDefinition(parser)
				if err != nil {
					return nil, err
				}
				nodes = append(nodes, node)
			case "union":
				node, err := parseUnionTypeDefinition(parser)
				if err != nil {
					return nil, err
				}
				nodes = append(nodes, node)
			case "scalar":
				node, err := parseScalarTypeDefinition(parser)
				if err != nil {
					return nil, err
				}
				nodes = append(nodes, node)
			case "enum":
				node, err := parseEnumTypeDefinition(parser)
				if err != nil {
					return nil, err
				}
				nodes = append(nodes, node)
			case "input":
				node, err := parseInputObjectTypeDefinition(parser)
				if err != nil {
					return nil, err
				}
				nodes = append(nodes, node)
			case "extend":
				node, err := parseTypeExtensionDefinition(parser)
				if err != nil {
					return nil, err
				}
				nodes = append(nodes, node)
			default:
				if err := unexpected(parser, lexer.Token{}); err != nil {
					return nil, err
				}
			}
		default:
			if err := unexpected(parser, lexer.Token{}); err != nil {
				return nil, err
			}
		}
	}
	return &ast.Document{
		Loc:         loc(parser, start),
		Definitions: nodes,
	}, nil
}

/* Implements the parsing rules in the Operations section. */

func parseOperationDefinition(parser *Parser) (*ast.OperationDefinition, error) {
	start := parser.Token.Start
	if peek(parser, lexer.BRACE_L) {
		selectionSet, err := parseSelectionSet(parser)
		if err != nil {
			return nil, err
		}
		return &ast.OperationDefinition{
			Operation:    "query",
			SelectionSet: selectionSet,
			Loc:          loc(parser, start),
		}, nil
	}
	operationToken, err := expect(parser, lexer.NAME)
	if err != nil {
		return nil, err
	}
	operation := operationToken.Value
	name, err := parseName(parser)
	if err != nil {
		return nil, err
	}
	variableDefinitions, err := parseVariableDefinitions(parser)
	if err != nil {
		return nil, err
	}
	directives, err := parseDirectives(parser)
	if err != nil {
		return nil, err
	}
	selectionSet, err := parseSelectionSet(parser)
	if err != nil {
		return nil, err
	}
	return &ast.OperationDefinition{
		Operation:           operation,
		Name:                name,
		VariableDefinitions: variableDefinitions,
		Directives:          directives,
		SelectionSet:        selectionSet,
		Loc:                 loc(parser, start),
	}, nil
}

func parseVariableDefinitions(parser *Parser) ([]*ast.VariableDefinition, error) {
	var variableDefinitions []*ast.VariableDefinition
	if peek(parser, lexer.PAREN_L) {
		vdefs, err := many(parser, lexer.PAREN_L, parseVariableDefinition, lexer.PAREN_R)
		if err != nil {
			return variableDefinitions, err
		}
		variableDefinitions := make([]*ast.VariableDefinition, 0, len(vdefs))
		for _, vdef := range vdefs {
			if vdef != nil {
				variableDefinitions = append(variableDefinitions, vdef.(*ast.VariableDefinition))
			}
		}
		return variableDefinitions, nil
	}
	return variableDefinitions, nil
}

func parseVariableDefinition(parser *Parser) (interface{}, error) {
	start := parser.Token.Start
	variable, err := parseVariable(parser)
	if err != nil {
		return nil, err
	}
	_, err = expect(parser, lexer.COLON)
	if err != nil {
		return nil, err
	}
	ttype, err := parseType(parser)
	if err != nil {
		return nil, err
	}
	var defaultValue ast.Value
	if skp, err := skip(parser, lexer.EQUALS); err != nil {
		return nil, err
	} else if skp {
		dv, err := parseValueLiteral(parser, true)
		if err != nil {
			return nil, err
		}
		defaultValue = dv
	}
	return &ast.VariableDefinition{
		Variable:     variable,
		Type:         ttype,
		DefaultValue: defaultValue,
		Loc:          loc(parser, start),
	}, nil
}

func parseVariable(parser *Parser) (*ast.Variable, error) {
	start := parser.Token.Start
	_, err := expect(parser, lexer.DOLLAR)
	if err != nil {
		return nil, err
	}
	name, err := parseName(parser)
	if err != nil {
		return nil, err
	}
	return &ast.Variable{
		Name: name,
		Loc:  loc(parser, start),
	}, nil
}

func parseSelectionSet(parser *Parser) (*ast.SelectionSet, error) {
	start := parser.Token.Start
	iSelections, err := many(parser, lexer.BRACE_L, parseSelection, lexer.BRACE_R)
	if err != nil {
		return nil, err
	}
	selections := make([]ast.Selection, 0, len(iSelections))
	for _, iSelection := range iSelections {
		if iSelection != nil {
			// type assert interface{} into Selection interface
			selections = append(selections, iSelection.(ast.Selection))
		}
	}

	return &ast.SelectionSet{
		Selections: selections,
		Loc:        loc(parser, start),
	}, nil
}

func parseSelection(parser *Parser) (interface{}, error) {
	if peek(parser, lexer.SPREAD) {
		r, err := parseFragment(parser)
		return r, err
	}
	return parseField(parser)
}

func parseField(parser *Parser) (*ast.Field, error) {
	start := parser.Token.Start
	nameOrAlias, err := parseName(parser)
	if err != nil {
		return nil, err
	}
	var (
		name  *ast.Name
		alias *ast.Name
	)
	skp, err := skip(parser, lexer.COLON)
	if err != nil {
		return nil, err
	} else if skp {
		alias = nameOrAlias
		name, err = parseName(parser)
		if err != nil {
			return nil, err
		}
	} else {
		name = nameOrAlias
	}
	arguments, err := parseArguments(parser)
	if err != nil {
		return nil, err
	}
	directives, err := parseDirectives(parser)
	if err != nil {
		return nil, err
	}
	var selectionSet *ast.SelectionSet
	if peek(parser, lexer.BRACE_L) {
		sSet, err := parseSelectionSet(parser)
		if err != nil {
			return nil, err
		}
		selectionSet = sSet
	}
	return &ast.Field{
		Alias:        alias,
		Name:         name,
		Arguments:    arguments,
		Directives:   directives,
		SelectionSet: selectionSet,
		Loc:          loc(parser, start),
	}, nil
}

func parseArguments(parser *Parser) ([]*ast.Argument, error) {
	var arguments []*ast.Argument
	if peek(parser, lexer.PAREN_L) {
		iArguments, err := many(parser, lexer.PAREN_L, parseArgument, lexer.PAREN_R)
		if err != nil {
			return arguments, err
		}
		arguments := make([]*ast.Argument, 0, len(iArguments))
		for _, iArgument := range iArguments {
			if iArgument != nil {
				arguments = append(arguments, iArgument.(*ast.Argument))
			}
		}
		return arguments, nil
	}
	return arguments, nil
}

func parseArgument(parser *Parser) (interface{}, error) {
	start := parser.Token.Start
	name, err := parseName(parser)
	if err != nil {
		return nil, err
	}
	_, err = expect(parser, lexer.COLON)
	if err != nil {
		return nil, err
	}
	value, err := parseValueLiteral(parser, false)
	if err != nil {
		return nil, err
	}
	return &ast.Argument{
		Name:  name,
		Value: value,
		Loc:   loc(parser, start),
	}, nil
}

/* Implements the parsing rules in the Fragments section. */

func parseFragment(parser *Parser) (interface{}, error) {
	start := parser.Token.Start
	if _, err := expect(parser, lexer.SPREAD); err != nil {
		return nil, err
	}
	if parser.Token.Value == "on" {
		if err := advance(parser); err != nil {
			return nil, err
		}
		name, err := parseNamed(parser)
		if err != nil {
			return nil, err
		}
		directives, err := parseDirectives(parser)
		if err != nil {
			return nil, err
		}
		selectionSet, err := parseSelectionSet(parser)
		if err != nil {
			return nil, err
		}
		return &ast.InlineFragment{
			TypeCondition: name,
			Directives:    directives,
			SelectionSet:  selectionSet,
			Loc:           loc(parser, start),
		}, nil
	}
	name, err := parseFragmentName(parser)
	if err != nil {
		return nil, err
	}
	directives, err := parseDirectives(parser)
	if err != nil {
		return nil, err
	}
	return &ast.FragmentSpread{
		Name:       name,
		Directives: directives,
		Loc:        loc(parser, start),
	}, nil
}

func parseFragmentDefinition(parser *Parser) (*ast.FragmentDefinition, error) {
	start := parser.Token.Start
	_, err := expectKeyWord(parser, "fragment")
	if err != nil {
		return nil, err
	}
	name, err := parseFragmentName(parser)
	if err != nil {
		return nil, err
	}
	_, err = expectKeyWord(parser, "on")
	if err != nil {
		return nil, err
	}
	typeCondition, err := parseNamed(parser)
	if err != nil {
		return nil, err
	}
	directives, err := parseDirectives(parser)
	if err != nil {
		return nil, err
	}
	selectionSet, err := parseSelectionSet(parser)
	if err != nil {
		return nil, err
	}
	return &ast.FragmentDefinition{
		Name:          name,
		TypeCondition: typeCondition,
		Directives:    directives,
		SelectionSet:  selectionSet,
		Loc:           loc(parser, start),
	}, nil
}

func parseFragmentName(parser *Parser) (*ast.Name, error) {
	if parser.Token.Value == "on" {
		return nil, unexpected(parser, lexer.Token{})
	}
	return parseName(parser)
}

/* Implements the parsing rules in the Values section. */

func parseValueLiteral(parser *Parser, isConst bool) (ast.Value, error) {
	token := parser.Token
	switch token.Kind {
	case lexer.BRACKET_L:
		return parseList(parser, isConst)
	case lexer.BRACE_L:
		return parseObject(parser, isConst)
	case lexer.INT:
		if err := advance(parser); err != nil {
			return nil, err
		}
		return &ast.IntValue{
			Value: token.Value,
			Loc:   loc(parser, token.Start),
		}, nil
	case lexer.FLOAT:
		if err := advance(parser); err != nil {
			return nil, err
		}
		return &ast.FloatValue{
			Value: token.Value,
			Loc:   loc(parser, token.Start),
		}, nil
	case lexer.STRING:
		if err := advance(parser); err != nil {
			return nil, err
		}
		return &ast.StringValue{
			Value: token.Value,
			Loc:   loc(parser, token.Start),
		}, nil
	case lexer.NAME:
		if token.Value == "true" || token.Value == "false" {
			if err := advance(parser); err != nil {
				return nil, err
			}
			value := true
			if token.Value == "false" {
				value = false
			}
			return &ast.BooleanValue{
				Value: value,
				Loc:   loc(parser, token.Start),
			}, nil
		} else if token.Value != "null" {
			if err := advance(parser); err != nil {
				return nil, err
			}
			return &ast.EnumValue{
				Value: token.Value,
				Loc:   loc(parser, token.Start),
			}, nil
		}
	case lexer.DOLLAR:
		if !isConst {
			return parseVariable(parser)
		}
	}
	return nil, unexpected(parser, lexer.Token{})
}

func parseConstValue(parser *Parser) (interface{}, error) {
	return parseValueLiteral(parser, true)
}

func parseValueValue(parser *Parser) (interface{}, error) {
	return parseValueLiteral(parser, false)
}

func parseList(parser *Parser, isConst bool) (*ast.ListValue, error) {
	start := parser.Token.Start
	var item parseFn
	if isConst {
		item = parseConstValue
	} else {
		item = parseValueValue
	}
	iValues, err := any(parser, lexer.BRACKET_L, item, lexer.BRACKET_R)
	if err != nil {
		return nil, err
	}
	values := make([]ast.Value, len(iValues))
	for i, v := range iValues {
		values[i] = v.(ast.Value)
	}
	return &ast.ListValue{
		Values: values,
		Loc:    loc(parser, start),
	}, nil
}

func parseObject(parser *Parser, isConst bool) (*ast.ObjectValue, error) {
	start := parser.Token.Start
	_, err := expect(parser, lexer.BRACE_L)
	if err != nil {
		return nil, err
	}
	var fields []*ast.ObjectField
	fieldNames := make(map[string]struct{})
	for {
		if skp, err := skip(parser, lexer.BRACE_R); err != nil {
			return nil, err
		} else if skp {
			break
		}
		field, fieldName, err := parseObjectField(parser, isConst, fieldNames)
		if err != nil {
			return nil, err
		}
		fieldNames[fieldName] = struct{}{}
		fields = append(fields, field)
	}
	return &ast.ObjectValue{
		Fields: fields,
		Loc:    loc(parser, start),
	}, nil
}

func parseObjectField(parser *Parser, isConst bool, fieldNames map[string]struct{}) (*ast.ObjectField, string, error) {
	start := parser.Token.Start
	name, err := parseName(parser)
	if err != nil {
		return nil, "", err
	}
	fieldName := name.Value
	if _, ok := fieldNames[fieldName]; ok {
		descp := fmt.Sprintf("Duplicate input object field %v.", fieldName)
		return nil, "", gqlerrors.NewSyntaxError(parser.Source, start, descp)
	}
	_, err = expect(parser, lexer.COLON)
	if err != nil {
		return nil, "", err
	}
	value, err := parseValueLiteral(parser, isConst)
	if err != nil {
		return nil, "", err
	}
	return &ast.ObjectField{
		Name:  name,
		Value: value,
		Loc:   loc(parser, start),
	}, fieldName, nil
}

/* Implements the parsing rules in the Directives section. */

func parseDirectives(parser *Parser) ([]*ast.Directive, error) {
	var directives []*ast.Directive
	for {
		if !peek(parser, lexer.AT) {
			break
		}
		directive, err := parseDirective(parser)
		if err != nil {
			return directives, err
		}
		directives = append(directives, directive)
	}
	return directives, nil
}

func parseDirective(parser *Parser) (*ast.Directive, error) {
	start := parser.Token.Start
	_, err := expect(parser, lexer.AT)
	if err != nil {
		return nil, err
	}
	name, err := parseName(parser)
	if err != nil {
		return nil, err
	}
	args, err := parseArguments(parser)
	if err != nil {
		return nil, err
	}
	return &ast.Directive{
		Name:      name,
		Arguments: args,
		Loc:       loc(parser, start),
	}, nil
}

/* Implements the parsing rules in the Types section. */

func parseType(parser *Parser) (ast.Type, error) {
	start := parser.Token.Start
	var ttype ast.Type
	if skp, err := skip(parser, lexer.BRACKET_L); err != nil {
		return nil, err
	} else if skp {
		t, err := parseType(parser)
		if err != nil {
			return t, err
		}
		ttype = t
		_, err = expect(parser, lexer.BRACKET_R)
		if err != nil {
			return ttype, err
		}
		ttype = &ast.List{
			Type: ttype,
			Loc:  loc(parser, start),
		}
	} else {
		name, err := parseNamed(parser)
		if err != nil {
			return ttype, err
		}
		ttype = name
	}
	if skp, err := skip(parser, lexer.BANG); err != nil {
		return nil, err
	} else if skp {
		ttype = &ast.NonNull{
			Type: ttype,
			Loc:  loc(parser, start),
		}
		return ttype, nil
	}
	return ttype, nil
}

func parseNamed(parser *Parser) (*ast.Named, error) {
	start := parser.Token.Start
	name, err := parseName(parser)
	if err != nil {
		return nil, err
	}
	return &ast.Named{
		Name: name,
		Loc:  loc(parser, start),
	}, nil
}

/* Implements the parsing rules in the Type Definition section. */

func parseObjectTypeDefinition(parser *Parser) (*ast.ObjectDefinition, error) {
	start := parser.Token.Start
	_, err := expectKeyWord(parser, "type")
	if err != nil {
		return nil, err
	}
	name, err := parseName(parser)
	if err != nil {
		return nil, err
	}
	interfaces, err := parseImplementsInterfaces(parser)
	if err != nil {
		return nil, err
	}
	iFields, err := any(parser, lexer.BRACE_L, parseFieldDefinition, lexer.BRACE_R)
	if err != nil {
		return nil, err
	}
	fields := make([]*ast.FieldDefinition, 0, len(iFields))
	for _, iField := range iFields {
		if iField != nil {
			fields = append(fields, iField.(*ast.FieldDefinition))
		}
	}
	return &ast.ObjectDefinition{
		Name:       name,
		Loc:        loc(parser, start),
		Interfaces: interfaces,
		Fields:     fields,
	}, nil
}

func parseImplementsInterfaces(parser *Parser) ([]*ast.Named, error) {
	var types []*ast.Named
	if parser.Token.Value == "implements" {
		if err := advance(parser); err != nil {
			return nil, err
		}
		for {
			ttype, err := parseNamed(parser)
			if err != nil {
				return types, err
			}
			types = append(types, ttype)
			if peek(parser, lexer.BRACE_L) {
				break
			}
		}
	}
	return types, nil
}

func parseFieldDefinition(parser *Parser) (interface{}, error) {
	start := parser.Token.Start
	name, err := parseName(parser)
	if err != nil {
		return nil, err
	}
	args, err := parseArgumentDefs(parser)
	if err != nil {
		return nil, err
	}
	_, err = expect(parser, lexer.COLON)
	if err != nil {
		return nil, err
	}
	ttype, err := parseType(parser)
	if err != nil {
		return nil, err
	}
	return &ast.FieldDefinition{
		Name:      name,
		Arguments: args,
		Type:      ttype,
		Loc:       loc(parser, start),
	}, nil
}

func parseArgumentDefs(parser *Parser) ([]*ast.InputValueDefinition, error) {
	if !peek(parser, lexer.PAREN_L) {
		return nil, nil
	}
	iInputValueDefinitions, err := many(parser, lexer.PAREN_L, parseInputValueDef, lexer.PAREN_R)
	if err != nil {
		return nil, err
	}
	inputValueDefinitions := make([]*ast.InputValueDefinition, 0, len(iInputValueDefinitions))
	for _, iInputValueDefinition := range iInputValueDefinitions {
		if iInputValueDefinition != nil {
			inputValueDefinitions = append(inputValueDefinitions, iInputValueDefinition.(*ast.InputValueDefinition))
		}
	}
	return inputValueDefinitions, err
}

func parseInputValueDef(parser *Parser) (interface{}, error) {
	start := parser.Token.Start
	name, err := parseName(parser)
	if err != nil {
		return nil, err
	}
	_, err = expect(parser, lexer.COLON)
	if err != nil {
		return nil, err
	}
	ttype, err := parseType(parser)
	if err != nil {
		return nil, err
	}
	var defaultValue ast.Value
	if skp, err := skip(parser, lexer.EQUALS); err != nil {
		return nil, err
	} else if skp {
		val, err := parseConstValue(parser)
		if err != nil {
			return nil, err
		}
		if val, ok := val.(ast.Value); ok {
			defaultValue = val
		}
	}
	return &ast.InputValueDefinition{
		Name:         name,
		Type:         ttype,
		DefaultValue: defaultValue,
		Loc:          loc(parser, start),
	}, nil
}

func parseInterfaceTypeDefinition(parser *Parser) (*ast.InterfaceDefinition, error) {
	start := parser.Token.Start
	_, err := expectKeyWord(parser, "interface")
	if err != nil {
		return nil, err
	}
	name, err := parseName(parser)
	if err != nil {
		return nil, err
	}
	iFields, err := any(parser, lexer.BRACE_L, parseFieldDefinition, lexer.BRACE_R)
	if err != nil {
		return nil, err
	}
	fields := make([]*ast.FieldDefinition, 0, len(iFields))
	for _, iField := range iFields {
		if iField != nil {
			fields = append(fields, iField.(*ast.FieldDefinition))
		}
	}
	return &ast.InterfaceDefinition{
		Name:   name,
		Loc:    loc(parser, start),
		Fields: fields,
	}, nil
}

func parseUnionTypeDefinition(parser *Parser) (*ast.UnionDefinition, error) {
	start := parser.Token.Start
	_, err := expectKeyWord(parser, "union")
	if err != nil {
		return nil, err
	}
	name, err := parseName(parser)
	if err != nil {
		return nil, err
	}
	_, err = expect(parser, lexer.EQUALS)
	if err != nil {
		return nil, err
	}
	types, err := parseUnionMembers(parser)
	if err != nil {
		return nil, err
	}
	return &ast.UnionDefinition{
		Name:  name,
		Loc:   loc(parser, start),
		Types: types,
	}, nil
}

func parseUnionMembers(parser *Parser) ([]*ast.Named, error) {
	var members []*ast.Named
	for {
		member, err := parseNamed(parser)
		if err != nil {
			return members, err
		}
		members = append(members, member)
		if skp, err := skip(parser, lexer.PIPE); err != nil {
			return nil, err
		} else if !skp {
			break
		}
	}
	return members, nil
}

func parseScalarTypeDefinition(parser *Parser) (*ast.ScalarDefinition, error) {
	start := parser.Token.Start
	_, err := expectKeyWord(parser, "scalar")
	if err != nil {
		return nil, err
	}
	name, err := parseName(parser)
	if err != nil {
		return nil, err
	}
	def := &ast.ScalarDefinition{
		Name: name,
		Loc:  loc(parser, start),
	}
	return def, nil
}

func parseEnumTypeDefinition(parser *Parser) (*ast.EnumDefinition, error) {
	start := parser.Token.Start
	_, err := expectKeyWord(parser, "enum")
	if err != nil {
		return nil, err
	}
	name, err := parseName(parser)
	if err != nil {
		return nil, err
	}
	iEnumValueDefs, err := any(parser, lexer.BRACE_L, parseEnumValueDefinition, lexer.BRACE_R)
	if err != nil {
		return nil, err
	}
	values := make([]*ast.EnumValueDefinition, 0, len(iEnumValueDefs))
	for _, iEnumValueDef := range iEnumValueDefs {
		if iEnumValueDef != nil {
			values = append(values, iEnumValueDef.(*ast.EnumValueDefinition))
		}
	}
	return &ast.EnumDefinition{
		Name:   name,
		Loc:    loc(parser, start),
		Values: values,
	}, nil
}

func parseEnumValueDefinition(parser *Parser) (interface{}, error) {
	start := parser.Token.Start
	name, err := parseName(parser)
	if err != nil {
		return nil, err
	}
	return &ast.EnumValueDefinition{
		Name: name,
		Loc:  loc(parser, start),
	}, nil
}

func parseInputObjectTypeDefinition(parser *Parser) (*ast.InputObjectDefinition, error) {
	start := parser.Token.Start
	_, err := expectKeyWord(parser, "input")
	if err != nil {
		return nil, err
	}
	name, err := parseName(parser)
	if err != nil {
		return nil, err
	}
	iInputValueDefinitions, err := any(parser, lexer.BRACE_L, parseInputValueDef, lexer.BRACE_R)
	if err != nil {
		return nil, err
	}
	fields := make([]*ast.InputValueDefinition, 0, len(iInputValueDefinitions))
	for _, iInputValueDefinition := range iInputValueDefinitions {
		if iInputValueDefinition != nil {
			fields = append(fields, iInputValueDefinition.(*ast.InputValueDefinition))
		}
	}
	return &ast.InputObjectDefinition{
		Name:   name,
		Loc:    loc(parser, start),
		Fields: fields,
	}, nil
}

func parseTypeExtensionDefinition(parser *Parser) (*ast.TypeExtensionDefinition, error) {
	start := parser.Token.Start
	_, err := expectKeyWord(parser, "extend")
	if err != nil {
		return nil, err
	}

	definition, err := parseObjectTypeDefinition(parser)
	if err != nil {
		return nil, err
	}
	return &ast.TypeExtensionDefinition{
		Loc:        loc(parser, start),
		Definition: definition,
	}, nil
}

/* Core parsing utility functions */

// Returns a location object, used to identify the place in
// the source that created a given parsed object.
func loc(parser *Parser, start int) ast.Location {
	if parser.Options.NoSource {
		return ast.Location{
			Start: start,
			End:   parser.PrevEnd,
		}
	}
	return ast.Location{
		Start:  start,
		End:    parser.PrevEnd,
		Source: parser.Source,
	}
}

// Moves the internal parser object to the next lexed token.
func advance(parser *Parser) error {
	prevEnd := parser.Token.End
	parser.PrevEnd = prevEnd
	parser.Lexer.Seek(prevEnd)
	token, err := nextToken(parser)
	if err != nil {
		return err
	}
	parser.Token = token
	return nil
}

// nextToken returns the next token from the lexer skipping over comments
func nextToken(parser *Parser) (lexer.Token, error) {
	for {
		tok, err := parser.Lexer.NextToken()
		if err != nil {
			return tok, err
		}
		if tok.Kind != lexer.COMMENT {
			return tok, nil
		}
	}
}

// Determines if the next token is of a given kind
func peek(parser *Parser, Kind int) bool {
	return parser.Token.Kind == Kind
}

// If the next token is of the given kind, return true after advancing
// the parser. Otherwise, do not change the parser state and return false.
func skip(parser *Parser, Kind int) (bool, error) {
	if parser.Token.Kind == Kind {
		return true, advance(parser)
	}
	return false, nil
}

// If the next token is of the given kind, return that token after advancing
// the parser. Otherwise, do not change the parser state and return false.
func expect(parser *Parser, kind int) (lexer.Token, error) {
	token := parser.Token
	if token.Kind == kind {
		return token, advance(parser)
	}
	descp := fmt.Sprintf("Expected %s, found %s", lexer.GetTokenKindDesc(kind), lexer.GetTokenDesc(token))
	return token, gqlerrors.NewSyntaxError(parser.Source, token.Start, descp)
}

// If the next token is a keyword with the given value, return that token after
// advancing the parser. Otherwise, do not change the parser state and return false.
func expectKeyWord(parser *Parser, value string) (lexer.Token, error) {
	token := parser.Token
	if token.Kind == lexer.NAME && token.Value == value {
		return token, advance(parser)
	}
	descp := fmt.Sprintf("Expected \"%s\", found %s", value, lexer.GetTokenDesc(token))
	return token, gqlerrors.NewSyntaxError(parser.Source, token.Start, descp)
}

// Helper function for creating an error when an unexpected lexed token
// is encountered.
func unexpected(parser *Parser, atToken lexer.Token) error {
	token := atToken
	if (token == lexer.Token{}) {
		token = parser.Token
	}
	description := fmt.Sprintf("Unexpected %v", lexer.GetTokenDesc(token))
	return gqlerrors.NewSyntaxError(parser.Source, token.Start, description)
}

// any returns a possibly empty list of parse nodes, determined by
// the parseFn. This list begins with a lex token of openKind
// and ends with a lex token of closeKind. Advances the parser
// to the next lex token after the closing token.
func any(parser *Parser, openKind int, parseFn parseFn, closeKind int) ([]interface{}, error) {
	if _, err := expect(parser, openKind); err != nil {
		return nil, err
	}
	var nodes []interface{}
	for {
		if skp, err := skip(parser, closeKind); err != nil {
			return nil, err
		} else if skp {
			break
		}
		n, err := parseFn(parser)
		if err != nil {
			return nodes, err
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}

// many returns a non-empty list of parse nodes, determined by
// the parseFn. This list begins with a lex token of openKind
// and ends with a lex token of closeKind. Advances the parser
// to the next lex token after the closing token.
func many(parser *Parser, openKind int, parseFn parseFn, closeKind int) ([]interface{}, error) {
	_, err := expect(parser, openKind)
	if err != nil {
		return nil, err
	}
	node, err := parseFn(parser)
	if err != nil {
		return nil, err
	}
	var nodes []interface{}
	nodes = append(nodes, node)
	for {
		if skp, err := skip(parser, closeKind); err != nil {
			return nil, err
		} else if skp {
			break
		}
		node, err := parseFn(parser)
		if err != nil {
			return nodes, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}
