package graphql

import (
	"github.com/sprucehealth/graphql/gqlerrors"
	"github.com/sprucehealth/graphql/language/ast"
)

func NewLocatedError(err interface{}, nodes []ast.Node) *gqlerrors.Error {
	message := "An unknown error occurred."
	if err, ok := err.(error); ok {
		message = err.Error()
	}
	if err, ok := err.(string); ok {
		message = err
	}
	stack := message
	return gqlerrors.NewError(
		message,
		nodes,
		stack,
		nil,
		[]int{},
	)
}

func FieldASTsToNodeASTs(fieldASTs []*ast.Field) []ast.Node {
	nodes := make([]ast.Node, len(fieldASTs))
	for i, fieldAST := range fieldASTs {
		nodes[i] = fieldAST
	}
	return nodes
}
