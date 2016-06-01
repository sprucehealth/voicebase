package main

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
)

var createCarePlanInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "CreateCarePlanInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"uuid":             newUUIDInputField(),
		"name":             &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"treatments":       &graphql.InputObjectFieldConfig{Type: graphql.NewList(carePlanTreatmentInputType)},
		"instructions":     &graphql.InputObjectFieldConfig{Type: graphql.NewList(carePlanInstructionInputType)},
	},
})

var carePlanTreatmentInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "CarePlanTreatmentInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"ePrescribe":           &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Boolean)},
		"name":                 &graphql.InputObjectFieldConfig{Type: graphql.String, Description: "required if ePrescribe is false, ignored otherwise"},
		"medicationID":         &graphql.InputObjectFieldConfig{Type: graphql.ID, Description: "as returned by medicationSearch, required if ePrescribe is true"},
		"dosage":               &graphql.InputObjectFieldConfig{Type: graphql.String, Description: "required if ePrescribe is true"},
		"dispenseType":         &graphql.InputObjectFieldConfig{Type: graphql.String, Description: "required if ePrescribe is true"},
		"dispenseNumber":       &graphql.InputObjectFieldConfig{Type: graphql.Int, Description: "required if ePrescribe is true"},
		"refills":              &graphql.InputObjectFieldConfig{Type: graphql.Int, Description: "required if ePrescribe is true"},
		"substitutionsAllowed": &graphql.InputObjectFieldConfig{Type: graphql.Boolean, Description: "required if ePrescribe is true"},
		"daysSupply":           &graphql.InputObjectFieldConfig{Type: graphql.Int},
		"sig":                  &graphql.InputObjectFieldConfig{Type: graphql.String, Description: "required if ePrescribe is true"},
		"pharmacyID":           &graphql.InputObjectFieldConfig{Type: graphql.ID, Description: "required if ePrescribe is true"},
		"pharmacyInstructions": &graphql.InputObjectFieldConfig{Type: graphql.String, Description: "required if ePrescribe is true"},
	},
})

var carePlanInstructionInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "CarePlanInstructionInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"title": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"steps": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.String)))},
	},
})

type createCarePlanInput struct {
	ClientMutationID string                      `gql:"clientMutationId"`
	UUID             string                      `gql:"uuid"`
	Name             string                      `gql:"name"`
	Treatments       []*carePlanTreatmentInput   `gql:"treatments"`
	Instructions     []*carePlanInstructionInput `gql:"instructions"`
}

type carePlanTreatmentInput struct {
	EPrescribe           bool   `gql:"ePrescribe"`
	Name                 string `gql:"name"`
	MedicationID         string `gql:"medicationID"`
	Dosage               string `gql:"dosage"`
	DispenseType         string `gql:"dispenseType"`
	DispenseNumber       int    `gql:"dispenseNumber"`
	Refills              int    `gql:"refills"`
	SubstitutionsAllowed bool   `gql:"substitutionsAllowed"`
	DaysSupply           int    `gql:"daysSupply"`
	Sig                  string `gql:"sig"`
	PharmacyID           string `gql:"pharmacyID"`
	PharmacyInstructions string `gql:"pharmacyInstructions"`
}

type carePlanInstructionInput struct {
	Title string   `gql:"title"`
	Steps []string `gql:"steps"`
}

const (
	createCarePlanErrorCodeInvalidField = "INVALID_FIELD"
)

var createCarePlanErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name:        "CreateCarePlanErrorCode",
	Description: "Result of createCarePlan mutation",
	Values: graphql.EnumValueConfigMap{
		createCarePlanErrorCodeInvalidField: &graphql.EnumValueConfig{
			Value:       createCarePlanErrorCodeInvalidField,
			Description: "A field in the input is invalid",
		},
	},
})

type createCarePlanOutput struct {
	ClientMutationID string           `json:"clientMutationId,omitempty"`
	Success          bool             `json:"success"`
	ErrorCode        string           `json:"errorCode,omitempty"`
	ErrorMessage     string           `json:"errorMessage,omitempty"`
	CarePlan         *models.CarePlan `json:"carePlan,omitempty"`
}

var createCarePlanOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "CreateCarePlanPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientMutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: createCarePlanErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
		"carePlan":         &graphql.Field{Type: carePlanType},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*createCarePlanOutput)
		return ok
	},
})

var createCarePlanMutation = &graphql.Field{
	Type: graphql.NewNonNull(createCarePlanOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(createCarePlanInputType)},
	},
	Resolve: apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		acc := gqlctx.Account(ctx)

		input := p.Args["input"].(map[string]interface{})
		var in createCarePlanInput
		if err := gqldecode.Decode(input, &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(ctx, err)
		}

		if in.Name == "" {
			return &createCarePlanOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        createCarePlanErrorCodeInvalidField,
				ErrorMessage:     "Please enter a name for the care plan.",
			}, nil
		}

		req := &care.CreateCarePlanRequest{
			Name:         in.Name,
			CreatorID:    acc.ID,
			Instructions: make([]*care.CarePlanInstruction, len(in.Instructions)),
			Treatments:   make([]*care.CarePlanTreatment, len(in.Treatments)),
		}
		for i, ins := range in.Instructions {
			req.Instructions[i] = &care.CarePlanInstruction{
				Title: ins.Title,
				Steps: ins.Steps,
			}
		}
		for i, tr := range in.Treatments {
			ct := &care.CarePlanTreatment{
				EPrescribe:           tr.EPrescribe,
				MedicationID:         tr.MedicationID,
				Name:                 tr.Name,
				Dosage:               tr.Dosage,
				DispenseType:         tr.DispenseType,
				DispenseNumber:       uint32(tr.DispenseNumber),
				Refills:              uint32(tr.Refills),
				SubstitutionsAllowed: tr.SubstitutionsAllowed,
				DaysSupply:           uint32(tr.DaysSupply),
				Sig:                  tr.Sig,
				PharmacyID:           tr.PharmacyID,
				PharmacyInstructions: tr.PharmacyInstructions,
			}
			// TODO: validate pharmacy ID
			// TODO: lookup medication by ID if provided to fill in form, route, name, etc..
			req.Treatments[i] = ct
		}
		res, err := ram.CreateCarePlan(ctx, req)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		cp, err := transformCarePlanToResponse(res.CarePlan)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		return &createCarePlanOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          true,
			CarePlan:         cp,
		}, nil
	}),
}
