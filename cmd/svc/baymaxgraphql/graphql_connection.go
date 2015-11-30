package main

import (
	"fmt"

	"github.com/graphql-go/graphql"
)

type ConnectionCursor string

type PageInfo struct {
	HasPreviousPage bool `json:"hasPreviousPage"`
	HasNextPage     bool `json:"hasNextPage"`
}

type Connection struct {
	Edges    []*Edge  `json:"edges"`
	PageInfo PageInfo `json:"pageInfo"`
}

func NewConnection() *Connection {
	return &Connection{
		Edges:    []*Edge{},
		PageInfo: PageInfo{},
	}
}

type Edge struct {
	Node   interface{}      `json:"node"`
	Cursor ConnectionCursor `json:"cursor"`
}

type ConnectionArguments struct {
	Before ConnectionCursor `json:"before"`
	After  ConnectionCursor `json:"after"`
	First  int              `json:"first"` // -1 for undefined, 0 would return zero results
	Last   int              `json:"last"`  //  -1 for undefined, 0 would return zero results
}

type ConnectionArgumentsConfig struct {
	Before ConnectionCursor `json:"before"`
	After  ConnectionCursor `json:"after"`

	// use pointers for `First` and `Last` fields
	// so constructor would know when to use default values
	First *int `json:"first"`
	Last  *int `json:"last"`
}

func ParseConnectionArguments(filters map[string]interface{}) ConnectionArguments {
	conn := ConnectionArguments{
		First:  -1,
		Last:   -1,
		Before: "",
		After:  "",
	}
	if filters != nil {
		if first, ok := filters["first"]; ok {
			if first, ok := first.(int); ok {
				conn.First = first
			}
		}
		if last, ok := filters["last"]; ok {
			if last, ok := last.(int); ok {
				conn.Last = last
			}
		}
		if before, ok := filters["before"]; ok {
			conn.Before = ConnectionCursor(fmt.Sprintf("%v", before))
		}
		if after, ok := filters["after"]; ok {
			conn.After = ConnectionCursor(fmt.Sprintf("%v", after))
		}
	}
	return conn
}

var baseConnectionArguments = graphql.FieldConfigArgument{
	"before": &graphql.ArgumentConfig{Type: graphql.String},
	"after":  &graphql.ArgumentConfig{Type: graphql.String},
	"first":  &graphql.ArgumentConfig{Type: graphql.Int},
	"last":   &graphql.ArgumentConfig{Type: graphql.Int},
}

func NewConnectionArguments(configMap graphql.FieldConfigArgument) graphql.FieldConfigArgument {
	if configMap == nil {
		configMap = graphql.FieldConfigArgument{}
	}
	for fieldName, argConfig := range baseConnectionArguments {
		configMap[fieldName] = argConfig
	}
	return configMap
}

type ConnectionConfig struct {
	Name             string          `json:"name"`
	NodeType         *graphql.Object `json:"nodeType"`
	EdgeFields       graphql.Fields  `json:"edgeFields"`
	ConnectionFields graphql.Fields  `json:"connectionFields"`
}

type EdgeType struct {
	Node   interface{}      `json:"node"`
	Cursor ConnectionCursor `json:"cursor"`
}
type GraphQLConnectionDefinitions struct {
	EdgeType       *graphql.Object `json:"edgeType"`
	ConnectionType *graphql.Object `json:"connectionType"`
}

var pageInfoType = graphql.NewObject(graphql.ObjectConfig{
	Name:        "PageInfo",
	Description: "Information about pagination in a connection.",
	Fields: graphql.Fields{
		"hasNextPage": &graphql.Field{
			Type:        graphql.NewNonNull(graphql.Boolean),
			Description: "When paginating forwards, signifies if there are more items",
		},
		"hasPreviousPage": &graphql.Field{
			Type:        graphql.NewNonNull(graphql.Boolean),
			Description: "When paginating backwards, signifies if there are more items",
		},
	},
})

func ConnectionDefinitions(config ConnectionConfig) *GraphQLConnectionDefinitions {
	edgeType := graphql.NewObject(graphql.ObjectConfig{
		Name:        config.Name + "Edge",
		Description: "An edge in a connection",
		Fields: graphql.Fields{
			"node": &graphql.Field{
				Type:        config.NodeType,
				Description: "The item at the end of the edge",
			},
			"cursor": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: " cursor for use in pagination",
			},
		},
	})
	for fieldName, fieldConfig := range config.EdgeFields {
		edgeType.AddFieldConfig(fieldName, fieldConfig)
	}

	connectionType := graphql.NewObject(graphql.ObjectConfig{
		Name:        config.Name + "Connection",
		Description: "A connection to a list of items.",

		Fields: graphql.Fields{
			"pageInfo": &graphql.Field{
				Type:        graphql.NewNonNull(pageInfoType),
				Description: "Information to aid in pagination.",
			},
			"edges": &graphql.Field{
				Type:        graphql.NewList(edgeType),
				Description: "Information to aid in pagination.",
			},
		},
	})
	for fieldName, fieldConfig := range config.ConnectionFields {
		connectionType.AddFieldConfig(fieldName, fieldConfig)
	}

	return &GraphQLConnectionDefinitions{
		EdgeType:       edgeType,
		ConnectionType: connectionType,
	}
}
