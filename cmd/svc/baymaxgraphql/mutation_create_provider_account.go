package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	segment "github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/analytics"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/caremessenger/deeplink"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/textutil"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/operational"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
	"google.golang.org/grpc"
)

const (
	supportThreadTitle    = "Spruce Support"
	onboardingThreadTitle = "Setup Assistant"
	teamSpruceInitialText = `This is a support conversation with Spruce Health.

If you're unsure about anything or need some help, send us a message here and a member of the Spruce Health team will respond.`
)

type createProviderAccountOutput struct {
	ClientMutationID    string         `json:"clientMutationId,omitempty"`
	Success             bool           `json:"success"`
	ErrorCode           string         `json:"errorCode,omitempty"`
	ErrorMessage        string         `json:"errorMessage,omitempty"`
	Token               string         `json:"token,omitempty"`
	Account             models.Account `json:"account,omitempty"`
	ClientEncryptionKey string         `json:"clientEncryptionKey,omitempty"`
}

type createProviderAccountInput struct {
	ClientMutationID       string `gql:"clientMutationId"`
	UUID                   string `gql:"uuid"`
	Email                  string `gql:"email,nonempty"`
	Password               string `gql:"password,nonempty"`
	PhoneNumber            string `gql:"phoneNumber,nonempty"`
	FirstName              string `gql:"firstName,nonempty"`
	LastName               string `gql:"lastName,nonempty"`
	ShortTitle             string `gql:"shortTitle"`
	LongTitle              string `gql:"longTitle"`
	OrganizationName       string `gql:"organizationName"`
	PhoneVerificationToken string `gql:"phoneVerificationToken,nonempty"`
	Duration               string `gql:"duration"`
}

var createProviderAccountInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "CreateProviderAccountInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId":       newClientMutationIDInputField(),
		"uuid":                   newUUIDInputField(),
		"email":                  &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"password":               &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"phoneNumber":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"firstName":              &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"lastName":               &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"shortTitle":             &graphql.InputObjectFieldConfig{Type: graphql.String},
		"longTitle":              &graphql.InputObjectFieldConfig{Type: graphql.String},
		"organizationName":       &graphql.InputObjectFieldConfig{Type: graphql.String},
		"phoneVerificationToken": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"duration":               &graphql.InputObjectFieldConfig{Type: tokenDurationEnum},
	},
})

var createProviderAccountOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "CreateProviderAccountPayload",
	Fields: graphql.Fields{
		"clientMutationId":    newClientMutationIDOutputField(),
		"success":             &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":           &graphql.Field{Type: createAccountErrorCodeEnum},
		"errorMessage":        &graphql.Field{Type: graphql.String},
		"token":               &graphql.Field{Type: graphql.String},
		"account":             &graphql.Field{Type: accountInterfaceType},
		"clientEncryptionKey": &graphql.Field{Type: graphql.String},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*createProviderAccountOutput)
		return ok
	},
})

var createProviderAccountMutation = &graphql.Field{
	Type: graphql.NewNonNull(createProviderAccountOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(createProviderAccountInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		return createProviderAccount(p)
	},
}

func createProviderAccount(p graphql.ResolveParams) (*createProviderAccountOutput, error) {
	svc := serviceFromParams(p)
	ram := raccess.ResourceAccess(p)
	ctx := p.Context

	// TODO: We shoudln't need to keep this map around, but need it for entityInfoFromInput currently
	input := p.Args["input"].(map[string]interface{})

	var in createProviderAccountInput
	if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
		switch err := err.(type) {
		case gqldecode.ErrValidationFailed:
			return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
		}
		return nil, errors.InternalError(ctx, err)
	}

	inv, attribValues, err := svc.inviteAndAttributionInfo(ctx)
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	// Sanity check to make sure we fail early in case we forgot to handle all new invite types
	if inv != nil && inv.Type != invite.LookupInviteResponse_COLLEAGUE {
		golog.Warningf("Device mapped to a %s invite attempting to create a Provider account. Ignoring invite.", inv.Type.String())
		inv = nil
	}

	req := &auth.CreateAccountRequest{
		Email:    in.Email,
		Password: in.Password,
		Type:     auth.AccountType_PROVIDER,
	}
	req.Email = strings.TrimSpace(req.Email)
	if !validate.Email(req.Email) {
		return &createProviderAccountOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          false,
			ErrorCode:        createAccountErrorCodeInvalidEmail,
			ErrorMessage:     "Please enter a valid email address.",
		}, nil
	}
	entityInfo, err := entityInfoFromInput(input)
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	req.FirstName = strings.TrimSpace(entityInfo.FirstName)
	req.LastName = strings.TrimSpace(entityInfo.LastName)
	if req.FirstName == "" || !textutil.IsValidPlane0Unicode(req.FirstName) {
		return &createProviderAccountOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          false,
			ErrorCode:        createAccountErrorCodeInvalidFirstName,
			ErrorMessage:     "Please enter a valid first name.",
		}, nil
	}
	if req.LastName == "" || !textutil.IsValidPlane0Unicode(req.LastName) {
		return &createProviderAccountOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          false,
			ErrorCode:        createAccountErrorCodeInvalidLastName,
			ErrorMessage:     "Please enter a valid last name.",
		}, nil
	}

	var organizationName string
	if inv == nil {
		organizationName = strings.TrimSpace(in.OrganizationName)
		if organizationName == "" || !textutil.IsValidPlane0Unicode(organizationName) {
			return &createProviderAccountOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        createAccountErrorCodeInvalidOrganizationName,
				ErrorMessage:     "Please enter a valid organization name.",
			}, nil
		}
	}
	verifiedValue, err := ram.VerifiedValue(ctx, in.PhoneVerificationToken)
	if grpc.Code(err) == auth.ValueNotYetVerified {
		return nil, errors.New("The phone number for the provided token has not yet been verified.")
	} else if err != nil {
		return nil, err
	}
	vpn, err := phone.ParseNumber(verifiedValue)
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}
	pn, err := phone.ParseNumber(in.PhoneNumber)
	if err != nil {
		return &createProviderAccountOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          false,
			ErrorCode:        createAccountErrorCodeInvalidPhoneNumber,
			ErrorMessage:     "Please enter a valid phone number.",
		}, nil
	}
	req.PhoneNumber = pn.String()
	if vpn.String() != pn.String() {
		golog.Debugf("The provided phone number %q does not match the number validated by the provided token %s", pn.String(), vpn.String())
		return nil, fmt.Errorf("The provided phone number %q does not match the number validated by the provided token", req.PhoneNumber)
	}
	if in.Duration == "" {
		in.Duration = auth.TokenDuration_SHORT.String()
	}
	req.Duration = auth.TokenDuration(auth.TokenDuration_value[in.Duration])
	res, err := ram.CreateAccount(ctx, req)
	if err != nil {
		switch grpc.Code(err) {
		case auth.DuplicateEmail:
			return &createProviderAccountOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        createAccountErrorCodeAccountExists,
				ErrorMessage:     "An account already exists with the entered email address.",
			}, nil
		case auth.InvalidEmail:
			return &createProviderAccountOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        createAccountErrorCodeInvalidEmail,
				ErrorMessage:     "Please enter a valid email address.",
			}, nil
		case auth.InvalidPhoneNumber:
			return &createProviderAccountOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        createAccountErrorCodeInvalidPhoneNumber,
				ErrorMessage:     "Please enter a valid phone number.",
			}, nil
		}
		return nil, errors.InternalError(ctx, err)
	}
	// TODO: updating the gqlctx this is safe for now because the GraphQL pkg serializes mutations.
	// that likely won't change, but this still isn't a great way to update the gqlctx.
	gqlctx.InPlaceWithAccount(ctx, res.Account)

	var orgEntityID string
	var accEntityID string
	var orgCreatedOperationalEvent *operational.NewOrgCreatedEvent

	{
		if inv == nil {
			// Create organization
			ent, err := ram.CreateEntity(ctx, &directory.CreateEntityRequest{
				EntityInfo: &directory.EntityInfo{
					GroupName:   organizationName,
					DisplayName: organizationName,
				},
				Type: directory.EntityType_ORGANIZATION,
				// For now map organizations into the root creating account so they can edit it.
				AccountID: res.Account.ID,
			})
			if err != nil {
				return nil, err
			}
			orgEntityID = ent.ID
			orgCreatedOperationalEvent = &operational.NewOrgCreatedEvent{
				OrgCreated: time.Now().Unix(),
			}
		} else {
			orgEntityID = inv.GetColleague().OrganizationEntityID
		}

		contacts := []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       req.PhoneNumber,
				Provisioned: false,
				Verified:    true,
			},
		}

		// Create entity
		ent, err := ram.CreateEntity(ctx, &directory.CreateEntityRequest{
			EntityInfo:                entityInfo,
			Type:                      directory.EntityType_INTERNAL,
			ExternalID:                res.Account.ID,
			InitialMembershipEntityID: orgEntityID,
			Contacts:                  contacts,
			AccountID:                 res.Account.ID,
		})
		if err != nil {
			return nil, err
		}
		accEntityID = ent.ID
	}

	// Create a default saved queries
	// TODO: make this more reliable & idempotent
	if err = ram.CreateSavedQuery(ctx, &threading.CreateSavedQueryRequest{
		OrganizationID: orgEntityID,
		EntityID:       accEntityID,
		Title:          "All",
		Query:          &threading.Query{},
		Ordinal:        1,
	}); err != nil {
		return nil, errors.InternalError(ctx, err)
	}
	if err = ram.CreateSavedQuery(ctx, &threading.CreateSavedQueryRequest{
		OrganizationID: orgEntityID,
		EntityID:       accEntityID,
		Title:          "Patient",
		Query:          &threading.Query{Expressions: []*threading.Expr{{Value: &threading.Expr_ThreadType_{ThreadType: threading.EXPR_THREAD_TYPE_PATIENT}}}},
		Ordinal:        2,
	}); err != nil {
		return nil, errors.InternalError(ctx, err)
	}
	if err = ram.CreateSavedQuery(ctx, &threading.CreateSavedQueryRequest{
		OrganizationID: orgEntityID,
		EntityID:       accEntityID,
		Title:          "Team",
		Query:          &threading.Query{Expressions: []*threading.Expr{{Value: &threading.Expr_ThreadType_{ThreadType: threading.EXPR_THREAD_TYPE_TEAM}}}},
		Ordinal:        3,
	}); err != nil {
		return nil, errors.InternalError(ctx, err)
	}
	if err = ram.CreateSavedQuery(ctx, &threading.CreateSavedQueryRequest{
		OrganizationID: orgEntityID,
		EntityID:       accEntityID,
		Title:          "@Pages",
		Query:          &threading.Query{Expressions: []*threading.Expr{{Value: &threading.Expr_Flag_{Flag: threading.EXPR_FLAG_UNREAD_REFERENCE}}}},
		Ordinal:        4,
	}); err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	var createLinkedThreadsResponse *threading.CreateLinkedThreadsResponse
	if inv == nil {
		// Create initial threads, but don't fail entirely on errors as this isn't critical to the account existing,
		// and because a hard fail leaves the account around but makes it look like it failed it's best just to
		// log and continue. Once the account creation is idempotent then can have this be a hard fail.

		// These are created synchronously to enforce strict ordering

		// Create a support thread (linked to Spruce support org) and the primary entities for them
		var tsEnt1, tsEnt2 *directory.Entity
		par := conc.NewParallel()
		par.Go(func() error {
			var err error
			tsEnt1, err = ram.CreateEntity(ctx, &directory.CreateEntityRequest{
				EntityInfo: &directory.EntityInfo{
					GroupName:   supportThreadTitle,
					DisplayName: supportThreadTitle,
				},
				Type: directory.EntityType_SYSTEM,
				InitialMembershipEntityID: orgEntityID,
			})
			return err
		})
		remoteSupportThreadTitle := fmt.Sprintf(supportThreadTitle+" (%s)", organizationName)
		par.Go(func() error {
			var err error
			tsEnt2, err = ram.CreateEntity(ctx, &directory.CreateEntityRequest{
				EntityInfo: &directory.EntityInfo{
					GroupName:   remoteSupportThreadTitle,
					DisplayName: remoteSupportThreadTitle,
				},
				Type: directory.EntityType_SYSTEM,
				InitialMembershipEntityID: svc.spruceOrgID,
			})
			return err
		})

		if err := par.Wait(); err != nil {
			golog.Errorf("Failed to create entity for support thread for org %s: %s", orgEntityID, err)
		} else {
			createLinkedThreadsResponse, err = ram.CreateLinkedThreads(ctx, &threading.CreateLinkedThreadsRequest{
				Organization1ID:      orgEntityID,
				Organization2ID:      svc.spruceOrgID,
				PrimaryEntity1ID:     tsEnt1.ID,
				PrimaryEntity2ID:     tsEnt2.ID,
				PrependSenderThread1: false,
				PrependSenderThread2: true,
				Summary:              supportThreadTitle + ": " + teamSpruceInitialText[:128],
				Text:                 teamSpruceInitialText,
				Type:                 threading.THREAD_TYPE_SUPPORT,
				SystemTitle1:         supportThreadTitle,
				SystemTitle2:         remoteSupportThreadTitle,
			})
			if err != nil {
				golog.Errorf("Failed to create linked support threads for org %s: %s", orgEntityID, err)
			}
		}

		// Create an onboarding thread and related system entity
		onbEnt, err := ram.CreateEntity(ctx, &directory.CreateEntityRequest{
			EntityInfo: &directory.EntityInfo{
				GroupName:   onboardingThreadTitle,
				DisplayName: onboardingThreadTitle,
			},
			Type: directory.EntityType_SYSTEM,
			InitialMembershipEntityID: orgEntityID,
		})
		if err != nil {
			golog.Errorf("Failed to create entity for onboarding thread for org %s: %s", orgEntityID, err)
		} else {
			_, err = ram.CreateOnboardingThread(ctx, &threading.CreateOnboardingThreadRequest{
				OrganizationID:  orgEntityID,
				PrimaryEntityID: onbEnt.ID,
			})
			if err != nil {
				golog.Errorf("Failed to create onboarding thread for org %s: %s", orgEntityID, err)
			}
		}
	}

	// Record analytics
	headers := devicectx.SpruceHeaders(ctx)
	var platform string
	if headers != nil {
		platform = headers.Platform.String()
		golog.Debugf("Provider Account created. ID = %s Device = %s", res.Account.ID, headers.DeviceID)
	}
	orgName := organizationName
	if inv != nil {
		oe, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: orgEntityID,
			},
			Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
		})
		if err != nil {
			golog.Errorf("Failed to lookup organization %s: %s", orgEntityID, err)
		} else {
			orgName = oe.Info.DisplayName
		}
	}
	conc.Go(func() {
		var supportLink string
		if createLinkedThreadsResponse != nil {
			supportLink = deeplink.ThreadURLShareable(svc.webDomain, svc.spruceOrgID, createLinkedThreadsResponse.Thread2.ID)
		}

		if orgCreatedOperationalEvent != nil {
			orgCreatedOperationalEvent.OrgSupportThreadID = createLinkedThreadsResponse.Thread1.ID
			orgCreatedOperationalEvent.SpruceSupportThreadID = createLinkedThreadsResponse.Thread2.ID
			orgCreatedOperationalEvent.InitialProviderEntityID = accEntityID

			if err := awsutil.PublishToSNSTopic(svc.sns, svc.supportMessageTopicARN, orgCreatedOperationalEvent); err != nil {
				golog.Errorf("Unable to publish to org event operational topic: %s", err.Error())
			}

		}
		userTraits := map[string]interface{}{
			"name":              res.Account.FirstName + " " + res.Account.LastName,
			"first_name":        res.Account.FirstName,
			"last_name":         res.Account.LastName,
			"phone":             req.PhoneNumber,
			"email":             req.Email,
			"title":             entityInfo.ShortTitle,
			"organization_name": orgName,
			"platform":          platform,
			"createdAt":         time.Now().Unix(),
			"type":              "provider",
		}
		groupTraits := map[string]interface{}{
			"name":         orgName,
			"support_link": supportLink,
		}
		eventProps := map[string]interface{}{
			"entity_id":       accEntityID,
			"organization_id": orgEntityID,
		}
		if inv != nil {
			eventProps["invite"] = inv.Type.String()
			userTraits["inviter_entity_id"] = inv.GetColleague().InviterEntityID
		}
		if s := attribValues["adjust_adgroup"]; s != "" {
			eventProps["adjust_adgroup"] = s
			userTraits["adjust_adgroup"] = s
			// Creating new org so likely we want to track source on it as well
			if inv == nil {
				groupTraits["adjust_adgroup"] = s
			}
		}
		analytics.SegmentIdentify(&segment.Identify{
			UserId: res.Account.ID,
			Traits: userTraits,
			Context: map[string]interface{}{
				"ip":        remoteAddrFromParams(p),
				"userAgent": userAgentFromParams(p),
			},
		})
		analytics.SegmentGroup(&segment.Group{
			UserId:  res.Account.ID,
			GroupId: orgEntityID,
			Traits:  groupTraits,
		})
		analytics.SegmentTrack(&segment.Track{
			Event:      "signedup",
			UserId:     res.Account.ID,
			Properties: eventProps,
		})
	})

	// Mark the invite as consumed if one was used
	if inv != nil && !ignorePhoneNumberCheckForInvite(inv) {
		conc.GoCtx(gqlctx.Clone(ctx), func(ctx context.Context) {
			if _, err := svc.invite.MarkInviteConsumed(ctx, &invite.MarkInviteConsumedRequest{Token: attribValues[inviteTokenAttributionKey]}); err != nil {
				golog.Errorf("Error while marking invite with code %q as consumed: %s", attribValues[inviteTokenAttributionKey], err)
			}
		})
	}

	result := p.Info.RootValue.(map[string]interface{})["result"].(*conc.Map)
	result.Set("auth_token", res.Token.Value)
	result.Set("auth_expiration", time.Unix(int64(res.Token.ExpirationEpoch), 0))

	return &createProviderAccountOutput{
		ClientMutationID:    in.ClientMutationID,
		Success:             true,
		Token:               res.Token.Value,
		Account:             transformAccountToResponse(res.Account),
		ClientEncryptionKey: res.Token.ClientEncryptionKey,
	}, nil
}
