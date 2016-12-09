package gql

import "github.com/sprucehealth/graphql"

// messageType is a type representing an message
var messageType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Message",
		Fields: graphql.Fields{
			"text":         &graphql.Field{Type: graphql.String},
			"title":        &graphql.Field{Type: graphql.String},
			"summary":      &graphql.Field{Type: graphql.String},
			"attachments":  &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(attachmentType))},
			"source":       &graphql.Field{Type: endpointType},
			"destinations": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(endpointType))},
			"textRefs":     &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(referenceType))},
		},
	})

// attachment is a type representing an attachment
var attachmentType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Attachment",
		Fields: graphql.Fields{
			"title":     &graphql.Field{Type: graphql.String},
			"url":       &graphql.Field{Type: graphql.String},
			"userTitle": &graphql.Field{Type: graphql.String},
			"contentID": &graphql.Field{Type: graphql.String},
		},
	})

// endpointType is a type representing an endpoint
var endpointType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Endpoint",
		Fields: graphql.Fields{
			"id":      &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"channel": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	})

// reference is a type representing an reference
var referenceType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Reference",
		Fields: graphql.Fields{
			"id":   &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"type": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	})
