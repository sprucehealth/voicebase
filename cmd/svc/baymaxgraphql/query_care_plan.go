package main

import (
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
)

var treatmentAvailabilityType = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "TreatmentAvailability",
		Description: "Availability of a medication",
		Values: graphql.EnumValueConfigMap{
			models.TreatmentAvailabilityUnknown: &graphql.EnumValueConfig{
				Value:       models.TreatmentAvailabilityUnknown,
				Description: "Unknown or unspecified",
			},
			models.TreatmentAvailabilityOTC: &graphql.EnumValueConfig{
				Value:       models.TreatmentAvailabilityOTC,
				Description: "Over-the-counter",
			},
			models.TreatmentAvailabilityRx: &graphql.EnumValueConfig{
				Value:       models.TreatmentAvailabilityRx,
				Description: "By prescription only",
			},
		},
	},
)

var carePlanType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "CarePlan",
		Interfaces: []*graphql.Interface{
			nodeInterfaceType,
		},
		Fields: graphql.Fields{
			"id":           &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"name":         &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"treatments":   &graphql.Field{Type: graphql.NewList(carePlanTreatmentType)},
			"instructions": &graphql.Field{Type: graphql.NewList(carePlanInstructionType)},
			"createdTimestamp": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Int),
				Description: "timestamp when the care plan was created",
			},
			"submitted": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"submittedTimestamp": &graphql.Field{
				Type:        graphql.Int,
				Description: "timestamp when the care plan was submitted (message with care plan as attachment is posted)",
			},
			"entity": &graphql.Field{
				Type: entityType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					ram := raccess.ResourceAccess(p)
					cp := p.Source.(*models.CarePlan)
					if cp.ParentID == "" {
						return nil, nil
					}
					// Make sure the parent is a thread item. Should never be anything else at the moment.
					if !strings.HasPrefix(cp.ParentID, threading.ThreadItemIDPrefix) {
						return nil, errors.InternalError(ctx, fmt.Errorf("Unsupported parent ID %s for care plan %s", cp.ParentID, cp.ID))
					}

					// Need to jump through some hoops to get to the entity ID, but we can easily cache this in the future.
					ti, err := ram.ThreadItem(ctx, cp.ParentID)
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}
					th, err := ram.Thread(ctx, ti.ThreadID, "")
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}
					if th.PrimaryEntityID == "" {
						// Shouldn't be possible as care plans are only attached to secure patient threads
						return nil, errors.InternalError(ctx, fmt.Errorf("No primary entity on thread %s for care plan %s", th.ID, cp.ID))
					}

					ent, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
						LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
						LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
							EntityID: th.PrimaryEntityID,
						},
						RequestedInformation: &directory.RequestedInformation{
							Depth:             0,
							EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
						},
						Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
						RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
					})
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}
					return ent, nil
				},
			},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*models.CarePlan)
			return ok
		},
	},
)

var carePlanTreatmentType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "CarePlanTreatment",
		Fields: graphql.Fields{
			"ePrescribe":           &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"name":                 &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"form":                 &graphql.Field{Type: graphql.String},
			"route":                &graphql.Field{Type: graphql.String},
			"availability":         &graphql.Field{Type: graphql.NewNonNull(treatmentAvailabilityType)},
			"dosage":               &graphql.Field{Type: graphql.String},
			"dispenseType":         &graphql.Field{Type: graphql.String},
			"dispenseNumber":       &graphql.Field{Type: graphql.Int},
			"refills":              &graphql.Field{Type: graphql.Int},
			"substitutionsAllowed": &graphql.Field{Type: graphql.Boolean},
			"daysSupply":           &graphql.Field{Type: graphql.Int},
			"sig":                  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"pharmacy":             &graphql.Field{Type: pharmacyType},
			"pharmacyInstructions": &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*models.CarePlanTreatment)
			return ok
		},
	},
)

var carePlanInstructionType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "CarePlanInstruction",
		Fields: graphql.Fields{
			"title": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"steps": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*models.CarePlanInstruction)
			return ok
		},
	},
)

var pharmacyType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Pharmacy",
		Fields: graphql.Fields{
			"id":              &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"name":            &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"address":         &graphql.Field{Type: graphql.NewNonNull(addressType)},
			"phoneNumber":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"retail":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"twentyFourHours": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"specialty":       &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"mailOrder":       &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*models.Pharmacy)
			return ok
		},
	},
)

var addressType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Address",
		Fields: graphql.Fields{
			"address1":  &graphql.Field{Type: graphql.String},
			"address2":  &graphql.Field{Type: graphql.String},
			"city":      &graphql.Field{Type: graphql.String},
			"stateCode": &graphql.Field{Type: graphql.String},
			"country":   &graphql.Field{Type: graphql.String},
			"zipCode":   &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*models.Address)
			return ok
		},
	},
)

var carePlanQuery = &graphql.Field{
	Type: graphql.NewNonNull(carePlanType),
	Args: graphql.FieldConfigArgument{
		"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		ram := raccess.ResourceAccess(p)
		id := p.Args["id"].(string)
		return lookupCarePlan(ctx, ram, id)
	},
}

func lookupCarePlan(ctx context.Context, ram raccess.ResourceAccessor, carePlanID string) (*models.CarePlan, error) {
	cp, err := ram.CarePlan(ctx, carePlanID)
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}
	return transformCarePlanToResponse(cp)
}
