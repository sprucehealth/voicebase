package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/smtpapi-go"
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/models"
	"github.com/sprucehealth/backend/libs/branch"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// go vet doesn't like that the first argument to grpcErrorf is not a string so alias the function with a different name :(
var grpcErrorf = grpc.Errorf

// unitTesting is set by the unit tests to force deterministic token generation
var unitTesting = false

const tokenLength = 16

type server struct {
	dal             dal.DAL
	clk             clock.Clock
	directoryClient directory.DirectoryClient
	branch          branch.Client
	sg              SendGridClient
	fromEmail       string
}

type popover struct {
	Title      string `json:"title"`
	Message    string `json:"message"`
	ButtonText string `json:"button_text"`
}

type organizationInvite struct {
	Popover popover `json:"popover"`
	OrgID   string  `json:"org_id"`
	OrgName string  `json:"org_name"`
}

type colleagueInviteClientData struct {
	OrganizationInvite organizationInvite `json:"organization_invite"`
}

// SendGridClient is the interface implemented by SendGrid clients
type SendGridClient interface {
	Send(*sendgrid.SGMail) error
}

// New returns an initialized instance of the invite server
func New(dal dal.DAL, clk clock.Clock, directoryClient directory.DirectoryClient, branch branch.Client, sg SendGridClient, fromEmail string) invite.InviteServer {
	if clk == nil {
		clk = clock.New()
	}
	return &server{
		dal:             dal,
		clk:             clk,
		directoryClient: directoryClient,
		branch:          branch,
		sg:              sg,
		fromEmail:       fromEmail,
	}
}

// AttributionData returns the attribution data for a device
func (s *server) AttributionData(ctx context.Context, in *invite.AttributionDataRequest) (*invite.AttributionDataResponse, error) {
	values, err := s.dal.AttributionData(ctx, in.DeviceID)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpcErrorf(codes.NotFound, fmt.Sprintf("No attribution data found for device ID '%s'", in.DeviceID))
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	out := &invite.AttributionDataResponse{
		Values: make([]*invite.AttributionValue, 0, len(values)),
	}
	for k, v := range values {
		out.Values = append(out.Values, &invite.AttributionValue{
			Key:   k,
			Value: v,
		})
	}
	return out, nil
}

// InviteColleagues sends invites to people to join an organization
func (s *server) InviteColleagues(ctx context.Context, in *invite.InviteColleaguesRequest) (*invite.InviteColleaguesResponse, error) {
	if in.OrganizationEntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "OrganizationEntityID is required")
	}
	if in.InviterEntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "InviterEntityID is required")
	}
	// Validate all colleague information
	for _, c := range in.Colleagues {
		if c.Email == "" {
			return nil, grpcErrorf(codes.InvalidArgument, "Email is required")
		}
		if !validate.Email(c.Email) {
			return nil, grpcErrorf(codes.InvalidArgument, fmt.Sprintf("Email '%s' is invalid", c.Email))
		}
		if c.PhoneNumber == "" {
			return nil, grpcErrorf(codes.InvalidArgument, "Phone number is required")
		}
		pn, err := phone.ParseNumber(c.PhoneNumber)
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, fmt.Sprintf("Phone number '%s' is invalid", c.PhoneNumber))
		}
		c.PhoneNumber = pn.String()
	}

	// Lookup organization to get name
	res, err := s.directoryClient.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: in.OrganizationEntityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
	})
	if err != nil {
		switch grpc.Code(err) {
		case codes.NotFound:
			return nil, grpcErrorf(codes.InvalidArgument, "OrganizationEntityID not found")
		}
		return nil, errors.Trace(err)
	}
	// Sanity check
	if len(res.Entities) != 1 {
		return nil, grpcErrorf(codes.Internal, fmt.Sprintf("Expected 1 organization got %d", len(res.Entities)))
	}
	if res.Entities[0].Type != directory.EntityType_ORGANIZATION {
		return nil, grpcErrorf(codes.InvalidArgument, "OrganizationEntityID not an organization")
	}
	org := res.Entities[0]

	// Lookup inviter to get name
	res, err = s.directoryClient.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: in.InviterEntityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
	})
	if err != nil {
		switch grpc.Code(err) {
		case codes.NotFound:
			return nil, grpcErrorf(codes.InvalidArgument, "InviterEntityID not found")
		}
		return nil, errors.Trace(err)
	}
	// Sanity check
	if len(res.Entities) != 1 {
		return nil, grpcErrorf(codes.Internal, fmt.Sprintf("Expected 1 entities got %d", len(res.Entities)))
	}
	if res.Entities[0].Type != directory.EntityType_INTERNAL {
		return nil, grpcErrorf(codes.InvalidArgument, "InviterEntityID not an internal entity")
	}
	inviter := res.Entities[0]

	inviteClientDataJSON, err := json.Marshal(colleagueInviteClientData{
		OrganizationInvite: organizationInvite{
			Popover: popover{
				Title:      "Welcome to Spruce!",
				Message:    inviter.Info.DisplayName + " has invited you to join them on Spruce.",
				ButtonText: "Get Started",
			},
			OrgID:   org.ID,
			OrgName: org.Info.DisplayName,
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	inviteClientDataStr := string(inviteClientDataJSON)

	for _, c := range in.Colleagues {
		// TODO: enqueue invite rather than sending directly
		var token, inviteURL string
		for retry := 0; retry < 5; retry++ {
			token, err = generateToken()
			if err != nil {
				return nil, errors.Trace(err)
			}
			inviteURL, err = s.branch.URL(map[string]interface{}{
				"invite_token": token,
				"client_data":  inviteClientDataStr,
			})
			if err != nil {
				golog.Errorf("Failed to generate branch URL: %s", err)
				continue
			}
			err = s.dal.InsertInvite(ctx, &models.Invite{
				Token:                token,
				Type:                 models.ColleagueInvite,
				OrganizationEntityID: in.OrganizationEntityID,
				InviterEntityID:      in.InviterEntityID,
				Email:                c.Email,
				PhoneNumber:          c.PhoneNumber,
				Created:              s.clk.Now(),
				URL:                  inviteURL,
			})
			if err == nil {
				break
			}
			if errors.Cause(err) != dal.ErrDuplicateInviteToken {
				golog.Errorf("Failed to insert invite: %s", err)
			}
		}
		if token == "" {
			continue
		}

		// TODO: use a template
		if err := s.sg.Send(&sendgrid.SGMail{
			To:      []string{c.Email},
			Subject: fmt.Sprintf("Invite to join %s", org.Info.DisplayName),
			Text: fmt.Sprintf(
				"I would like you to join my organization %s\n%s\n\nBest,\n%s",
				org.Info.DisplayName, inviteURL, inviter.Info.DisplayName),
			From:     s.fromEmail,
			FromName: inviter.Info.DisplayName,
			SMTPAPIHeader: smtpapi.SMTPAPIHeader{
				UniqueArgs: map[string]string{
					"invite_token": token,
				},
			},
		}); err != nil {
			golog.Errorf("Failed to send invite %s email to %s: %s", token, c.Email, err)
		}
	}
	return &invite.InviteColleaguesResponse{}, nil
}

// LookupInvite returns information about an invite by token
func (s *server) LookupInvite(ctx context.Context, in *invite.LookupInviteRequest) (*invite.LookupInviteResponse, error) {
	inv, err := s.dal.InviteForToken(ctx, in.Token)
	if err != nil {
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "Invite not found with token "+in.Token)
		}
		return nil, errors.Trace(err)
	}
	if inv.Type != models.ColleagueInvite {
		return nil, grpcErrorf(codes.Internal, "unsupported invite type "+string(inv.Type))
	}
	return &invite.LookupInviteResponse{
		Type: invite.LookupInviteResponse_COLLEAGUE,
		Invite: &invite.LookupInviteResponse_Colleague{
			Colleague: &invite.ColleagueInvite{
				OrganizationEntityID: inv.OrganizationEntityID,
				InviterEntityID:      inv.InviterEntityID,
				Colleague: &invite.Colleague{
					Email:       inv.Email,
					PhoneNumber: inv.PhoneNumber,
				},
			},
		},
	}, nil
}

// SetAttributionData associate attribution data with a device
func (s *server) SetAttributionData(ctx context.Context, in *invite.SetAttributionDataRequest) (*invite.SetAttributionDataResponse, error) {
	if in.DeviceID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "DeviceID is required")
	}
	values := make(map[string]string, len(in.Values))
	for _, v := range in.Values {
		values[v.Key] = v.Value
	}
	if err := s.dal.SetAttributionData(ctx, in.DeviceID, values); err != nil {
		return nil, errors.Trace(err)
	}
	return &invite.SetAttributionDataResponse{}, nil
}

func generateToken() (string, error) {
	if unitTesting {
		return "thetoken", nil
	}
	b := make([]byte, tokenLength)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", errors.Trace(err)
	}
	return strings.TrimRight(base64.URLEncoding.EncodeToString(b), "="), nil
}
