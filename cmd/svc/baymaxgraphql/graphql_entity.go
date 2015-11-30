package main

import "github.com/graphql-go/graphql"

var contactEnumType = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "ContactType",
		Description: "Type of contact value",
		Values: graphql.EnumValueConfigMap{
			"APP": &graphql.EnumValueConfig{
				Value:       "APP",
				Description: "Application or web",
			},
			"PHONE": &graphql.EnumValueConfig{
				Value:       "PHONE",
				Description: "Phone",
			},
			"EMAIL": &graphql.EnumValueConfig{
				Value:       "EMAIL",
				Description: "Email",
			},
		},
	},
)

var contactInfoType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ContactInfo",
		Fields: graphql.Fields{
			"type":        &graphql.Field{Type: graphql.NewNonNull(contactEnumType)},
			"value":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"provisioned": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		},
	},
)

var entityType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Entity",
		Interfaces: []*graphql.Interface{
			nodeInterfaceType,
		},
		Fields: graphql.Fields{
			"id":       &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"name":     &graphql.Field{Type: graphql.String},
			"contacts": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(contactInfoType))},
			// TODO: avatar(width: Int = 120, height: Int = 120, crop: Boolean = true): Image
		},
	},
)
