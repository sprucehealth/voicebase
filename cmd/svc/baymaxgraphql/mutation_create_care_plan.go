package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/graphql"
	"unicode/utf8"
)

var createCarePlanInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "CreateCarePlanInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"uuid":             newUUIDInputField(),
		"name":             &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"treatments":       &graphql.InputObjectFieldConfig{Type: graphql.NewList(carePlanTreatmentInput)},
		"instructions":     &graphql.InputObjectFieldConfig{Type: graphql.NewList(carePlanInstructionInput)},
	},
})

var carePlanTreatmentInput = graphql.NewInputObject(graphql.InputObjectConfig{
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

var carePlanInstructionInput = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "CarePlanInstructionInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"title": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"steps": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.String)))},
	},
})

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
		mutationID, _ := input["clientMutationId"].(string)

		name := input["name"].(string)
		treatmentsIn, _ := input["treatments"].([]interface{})
		instructionsIn, _ := input["instructions"].([]interface{})

		if name == "" {
			return &createCarePlanOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        createCarePlanErrorCodeInvalidField,
				ErrorMessage:     "Please enter a name for the care plan.",
			}, nil
		}
		if !utf8.ValidString(name) {
			return &createCarePlanOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        createCarePlanErrorCodeInvalidField,
				ErrorMessage:     "The entered name is not valid.",
			}, nil
		}

		req := &care.CreateCarePlanRequest{
			Name:         name,
			CreatorID:    acc.ID,
			Instructions: make([]*care.CarePlanInstruction, len(instructionsIn)),
			Treatments:   make([]*care.CarePlanTreatment, len(treatmentsIn)),
		}
		for i, in := range instructionsIn {
			inMap := in.(map[string]interface{})
			steps, _ := inMap["steps"].([]interface{})
			cin := &care.CarePlanInstruction{
				Title: inMap["title"].(string),
				Steps: make([]string, len(steps)),
			}
			for j, s := range steps {
				cin.Steps[j] = s.(string)
			}
			req.Instructions[i] = cin
		}
		for i, in := range treatmentsIn {
			inMap := in.(map[string]interface{})
			ct := &care.CarePlanTreatment{
				EPrescribe: inMap["ePrescribe"].(bool),
			}
			ct.MedicationID, _ = inMap["medicationID"].(string)
			ct.Name, _ = inMap["name"].(string)
			ct.Dosage, _ = inMap["dosage"].(string)
			ct.DispenseType, _ = inMap["dispenseType"].(string)
			dispenseNumber, _ := inMap["dispenseNumber"].(int)
			ct.DispenseNumber = uint32(dispenseNumber)
			refills, _ := inMap["refills"].(int)
			ct.Refills = uint32(refills)
			ct.SubstitutionsAllowed, _ = inMap["substitutionsAllowed"].(bool)
			daysSupply, _ := inMap["daysSupply"].(int)
			ct.DaysSupply = uint32(daysSupply)
			ct.Sig, _ = inMap["sig"].(string)
			ct.PharmacyID, _ = inMap["pharmacyID"].(string)
			ct.PharmacyInstructions, _ = inMap["pharmacyInstructions"].(string)
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
			ClientMutationID: mutationID,
			Success:          true,
			CarePlan:         cp,
		}, nil
	}),
}
