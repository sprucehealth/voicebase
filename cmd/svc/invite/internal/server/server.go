package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/smtpapi-go"
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/libs/branch"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/events"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/invite"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// go vet doesn't like that the first argument to grpcErrorf is not a string so alias the function with a different name :(
var grpcErrorf = grpc.Errorf

var complexTokenGenerator common.TokenGenerator

const complexTokenLength = 16

type complexTokenGen struct{}

const phiAttributeText = "PROTECTED_PHI"

func (t *complexTokenGen) GenerateToken() (string, error) {
	b := make([]byte, complexTokenLength)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", errors.Trace(err)
	}
	return strings.TrimRight(base64.URLEncoding.EncodeToString(b), "="), nil
}

var simpleTokenGenerator common.TokenGenerator

const simpleTokenLength = 6
const simpleTokenMaxValue = 999999

type simpleTokenGen struct{}

func (t *simpleTokenGen) GenerateToken() (string, error) {
	code, err := common.GenerateRandomNumber(simpleTokenMaxValue, simpleTokenLength)
	if err != nil {
		return "", errors.Trace(err)
	}
	return code, nil
}

func init() {
	simpleTokenGenerator = &simpleTokenGen{}
	complexTokenGenerator = &complexTokenGen{}
}

type server struct {
	dal             dal.DAL
	clk             clock.Clock
	directoryClient directory.DirectoryClient
	excommsClient   excomms.ExCommsClient
	branch          branch.Client
	sg              SendGridClient
	fromEmail       string
	fromNumber      string
	eventsTopic     string
	sns             snsiface.SNSAPI
	webInviteURL    *url.URL
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

type greeting struct {
	Title      string `json:"title"`
	Message    string `json:"message"`
	ButtonText string `json:"button_text"`
}

type patientInvite struct {
	Greeting greeting `json:"greeting"`
	OrgID    string   `json:"org_id"`
	OrgName  string   `json:"org_name"`
}

type patientInviteClientData struct {
	PatientInvite patientInvite `json:"patient_invite"`
}

// SendGridClient is the interface implemented by SendGrid clients
type SendGridClient interface {
	Send(*sendgrid.SGMail) error
}

// New returns an initialized instance of the invite server
func New(
	dal dal.DAL,
	clk clock.Clock,
	directoryClient directory.DirectoryClient,
	excommsClient excomms.ExCommsClient,
	snsC snsiface.SNSAPI,
	branch branch.Client,
	sg SendGridClient,
	fromEmail, fromNumber, eventsTopic, webInviteURL string) invite.InviteServer {
	if clk == nil {
		clk = clock.New()
	}
	var webURL *url.URL
	if webInviteURL != "" {
		var err error
		webURL, err = url.Parse(webInviteURL)
		if err != nil {
			golog.Fatalf("Failed to parse web invite URL: %s", err)
		}
	}
	return &server{
		dal:             dal,
		clk:             clk,
		directoryClient: directoryClient,
		excommsClient:   excommsClient,
		sns:             snsC,
		branch:          branch,
		sg:              sg,
		fromEmail:       fromEmail,
		fromNumber:      fromNumber,
		eventsTopic:     eventsTopic,
		webInviteURL:    webURL,
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

	// Lookup org to get name
	org, err := s.getOrg(ctx, in.OrganizationEntityID)
	if err != nil {
		return nil, err
	}

	// Lookup inviter to get name
	inviter, err := s.getInternalEntity(ctx, in.InviterEntityID)
	if err != nil {
		return nil, err
	}

	inviteClientDataJSON, err := json.Marshal(colleagueInviteClientData{
		OrganizationInvite: organizationInvite{
			Popover: popover{
				Title:      "Welcome to Spruce!",
				Message:    inviter.Info.DisplayName + " has invited you to join them on Spruce.",
				ButtonText: "Okay",
			},
			OrgID:   org.ID,
			OrgName: org.Info.DisplayName,
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	for _, c := range in.Colleagues {
		s.proccessInvite(
			ctx,
			complexTokenGenerator,
			org, inviter,
			"", "", c.Email, c.PhoneNumber, string(inviteClientDataJSON),
			models.ColleagueInvite, nil)
	}

	events.Publish(s.sns, s.eventsTopic, events.Service_INVITE, &invite.Event{
		Type: invite.Event_INVITED_COLLEAGUES,
		Details: &invite.Event_InvitedColleagues{
			InvitedColleagues: &invite.InvitedColleagues{
				OrganizationEntityID: in.OrganizationEntityID,
				InviterEntityID:      in.InviterEntityID,
			},
		},
	})

	return &invite.InviteColleaguesResponse{}, nil
}

// InvitePatients sends invites to people to join an organization
func (s *server) InvitePatients(ctx context.Context, in *invite.InvitePatientsRequest) (*invite.InvitePatientsResponse, error) {
	if in.OrganizationEntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "OrganizationEntityID is required")
	}
	if in.InviterEntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "InviterEntityID is required")
	}
	// Validate all colleague information
	for _, c := range in.Patients {
		if c.PhoneNumber == "" {
			return nil, grpcErrorf(codes.InvalidArgument, "Phone number is required")
		}
		pn, err := phone.ParseNumber(c.PhoneNumber)
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, fmt.Sprintf("Phone number '%s' is invalid", c.PhoneNumber))
		}
		c.PhoneNumber = pn.String()
	}

	// Lookup org to get name
	org, err := s.getOrg(ctx, in.OrganizationEntityID)
	if err != nil {
		return nil, err
	}

	// Lookup inviter to get name
	inviter, err := s.getInternalEntity(ctx, in.InviterEntityID)
	if err != nil {
		return nil, err
	}

	for _, p := range in.Patients {
		welcomeText := "Welcome!"
		if p.FirstName != "" {
			welcomeText = fmt.Sprintf("Welcome %s!", p.FirstName)
		}
		inviteClientDataJSON, err := json.Marshal(patientInviteClientData{
			PatientInvite: patientInvite{
				Greeting: greeting{
					Title:      welcomeText,
					Message:    fmt.Sprintf("Let's create your account so you can start securely messaging with %s.", org.Info.DisplayName),
					ButtonText: "Get Started",
				},
				OrgID:   org.ID,
				OrgName: org.Info.DisplayName,
			},
		})
		if err != nil {
			return nil, errors.Trace(err)
		}

		s.proccessInvite(
			ctx,
			simpleTokenGenerator,
			org, inviter,
			p.FirstName, p.ParkedEntityID, "", p.PhoneNumber, string(inviteClientDataJSON),
			models.PatientInvite,
			nil)
	}

	events.Publish(s.sns, s.eventsTopic, events.Service_INVITE, &invite.Event{
		Type: invite.Event_INVITED_PATIENTS,
		Details: &invite.Event_InvitedPatients{
			InvitedPatients: &invite.InvitedPatients{
				OrganizationEntityID: in.OrganizationEntityID,
				InviterEntityID:      in.InviterEntityID,
			},
		},
	})
	return &invite.InvitePatientsResponse{}, err
}

func (s *server) proccessInvite(
	ctx context.Context,
	tokenGenerator common.TokenGenerator,
	org, inviter *directory.Entity,
	firstName, parkedEntityID, email, phoneNumber, inviteClientDataStr string,
	inviteType models.InviteType,
	additionalValues map[string]string) error {
	// TODO: enqueue invite rather than sending directly
	var token, inviteURL string
	var err error
	for retry := 0; retry < 5; retry++ {
		token, err = tokenGenerator.GenerateToken()
		if err != nil {
			return errors.Trace(err)
		}
		values := map[string]string{
			"invite_token": token,
			"client_data":  inviteClientDataStr,
		}
		for k, v := range additionalValues {
			values[k] = v
		}
		if s.webInviteURL != nil {
			// Close the URL to avoid modifying the template
			ur := *s.webInviteURL
			query := ur.Query()
			query.Add("invite", token)
			ur.RawQuery = query.Encode()
			values[branch.DesktopURL] = ur.String()
		}
		attr := make(map[string]interface{}, len(values))
		for k, v := range values {
			attr[k] = v
		}
		inviteURL, err = s.branch.URL(attr)
		if err != nil {
			golog.Errorf("Failed to generate branch URL: %s", err)
			continue
		}
		pn := phoneNumber
		// We do not store phone numbers or emails for patients in dynamodb since it doesn't support simple encryption at rest
		if inviteType == models.PatientInvite {
			// We can't have these being empty attributes so populate them with informative info
			pn = phiAttributeText
			email = phiAttributeText
		}
		err = s.dal.InsertInvite(ctx, &models.Invite{
			Token:                token,
			Type:                 inviteType,
			OrganizationEntityID: org.ID,
			InviterEntityID:      inviter.ID,
			Email:                email,
			PhoneNumber:          pn,
			Created:              s.clk.Now(),
			URL:                  inviteURL,
			ParkedEntityID:       parkedEntityID,
			Values:               values,
		})
		if err == nil {
			break
		}
		if errors.Cause(err) != dal.ErrDuplicateInviteToken {
			golog.Errorf("Failed to insert invite: %s", err)
			return nil
		}
	}

	switch inviteType {
	case models.ColleagueInvite:
		if err := s.sendColleagueOutbound(ctx, email, inviteURL, token, org, inviter); err != nil {
			golog.Errorf("Failed to send colleague invite outbound comms: %s", err)
		}
	case models.PatientInvite:
		if err := s.sendPatientOutbound(ctx, firstName, phoneNumber, inviteURL, token, org, inviter); err != nil {
			golog.Errorf("Failed to send colleague invite outbound comms: %s", err)
		}
	default:
		golog.Errorf("Unknown invite type %s. No outbound message sent.", inviteType)
	}

	return nil
}

func (s *server) sendColleagueOutbound(ctx context.Context, email, inviteURL, token string, org, inviter *directory.Entity) error {
	// TODO: use a template
	err := s.sg.Send(&sendgrid.SGMail{
		To:      []string{email},
		Subject: fmt.Sprintf("Invite to join %s on Spruce", org.Info.DisplayName),
		Text: fmt.Sprintf(
			"Spruce is a communication and digital care app. By joining %s on Spruce, you'll be able to collaborate with colleagues around your patients' care, securely and efficiently.\n\nClick this link to get started:\n%s\n\nOnce you've created your account, you're all set to start catching up on the latest conversation.\n\nIf you have any troubles, we're here to help - simply reply to this email!\n\nThanks,\nThe Team at Spruce\n\nP.S.: Learn more about Spruce here: https://www.sprucehealth.com",
			org.Info.DisplayName, inviteURL),
		From:     s.fromEmail,
		FromName: inviter.Info.DisplayName,
		SMTPAPIHeader: smtpapi.SMTPAPIHeader{
			UniqueArgs: map[string]string{
				"invite_token": token,
			},
		},
	})
	if err != nil {
		golog.Errorf("Failed to send invite %s email to %s: %s", token, email, err)
	}
	return errors.Trace(err)
}

func (s *server) sendPatientOutbound(ctx context.Context, firstName, phoneNumber, inviteURL, token string, org, inviter *directory.Entity) error {
	golog.Debugf("Sending outbound patient invite messaging. URL: %s, Token: %s", inviteURL, token)
	msgText := fmt.Sprintf("%s has invited you to use Spruce for secure messaging and digital care.", org.Info.DisplayName)
	if firstName != "" {
		msgText = fmt.Sprintf("%s - ", firstName) + msgText
	}
	if _, err := s.excommsClient.SendMessage(ctx, &excomms.SendMessageRequest{
		Channel: excomms.ChannelType_SMS,
		Message: &excomms.SendMessageRequest_SMS{
			SMS: &excomms.SMSMessage{
				Text:            msgText,
				FromPhoneNumber: s.fromNumber,
				ToPhoneNumber:   phoneNumber,
			},
		},
	}); err != nil {
		return errors.Trace(err)
	}
	conc.AfterFunc(time.Second*1, func() {
		msgText = fmt.Sprintf("Get the Spruce app now and join them. %s [%s]", inviteURL, token)
		if _, err := s.excommsClient.SendMessage(context.Background(), &excomms.SendMessageRequest{
			Channel: excomms.ChannelType_SMS,
			Message: &excomms.SendMessageRequest_SMS{
				SMS: &excomms.SMSMessage{
					Text:            msgText,
					FromPhoneNumber: s.fromNumber,
					ToPhoneNumber:   phoneNumber,
				},
			},
		}); err != nil {
			golog.Errorf("Encountered an error while sending patient invite SMS: %s", err)
		}
	})
	return nil
}

func (s *server) getEntity(ctx context.Context, entityID string) (*directory.Entity, error) {
	// Lookup organization to get name
	res, err := s.directoryClient.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: entityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
	})
	if err != nil {
		switch grpc.Code(err) {
		case codes.NotFound:
			return nil, grpcErrorf(codes.InvalidArgument, "EntityID not found")
		}
		return nil, errors.Trace(err)
	}
	// Sanity check
	if len(res.Entities) != 1 {
		return nil, grpcErrorf(codes.Internal, fmt.Sprintf("Expected 1 entity got %d", len(res.Entities)))
	}
	return res.Entities[0], nil
}

func (s *server) getInternalEntity(ctx context.Context, entityID string) (*directory.Entity, error) {
	entity, err := s.getEntity(ctx, entityID)
	if err != nil {
		return nil, err
	}
	if entity.Type != directory.EntityType_INTERNAL {
		return nil, grpcErrorf(codes.InvalidArgument, "entityID not an internal entity")
	}
	return entity, nil
}

func (s *server) getOrg(ctx context.Context, orgID string) (*directory.Entity, error) {
	entity, err := s.getEntity(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if entity.Type != directory.EntityType_ORGANIZATION {
		return nil, grpcErrorf(codes.InvalidArgument, "OrganizationEntityID not an organization")
	}
	return entity, nil
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
	if inv.Type != models.ColleagueInvite && inv.Type != models.PatientInvite {
		return nil, grpcErrorf(codes.Internal, "unsupported invite type "+string(inv.Type))
	}
	values := make([]*invite.AttributionValue, 0, len(inv.Values))
	for k, v := range inv.Values {
		values = append(values, &invite.AttributionValue{
			Key:   k,
			Value: v,
		})
	}
	resp := &invite.LookupInviteResponse{Values: values}
	switch inv.Type {
	case models.ColleagueInvite:
		resp.Type = invite.LookupInviteResponse_COLLEAGUE
		resp.Invite = &invite.LookupInviteResponse_Colleague{
			Colleague: &invite.ColleagueInvite{
				OrganizationEntityID: inv.OrganizationEntityID,
				InviterEntityID:      inv.InviterEntityID,
				Colleague: &invite.Colleague{
					Email:       inv.Email,
					PhoneNumber: inv.PhoneNumber,
				},
			},
		}
	case models.PatientInvite:
		resp.Type = invite.LookupInviteResponse_PATIENT
		resp.Invite = &invite.LookupInviteResponse_Patient{
			Patient: &invite.PatientInvite{
				OrganizationEntityID: inv.OrganizationEntityID,
				InviterEntityID:      inv.InviterEntityID,
				Patient: &invite.Patient{
					ParkedEntityID: inv.ParkedEntityID,
				},
			},
		}
	}
	return resp, nil
}

// MarkInviteConsumed deletes the associated invite and records it's consumption
func (s *server) MarkInviteConsumed(ctx context.Context, in *invite.MarkInviteConsumedRequest) (*invite.MarkInviteConsumedResponse, error) {
	// TODO: Record consumption metrics
	if err := s.dal.DeleteInvite(ctx, in.Token); err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	return &invite.MarkInviteConsumedResponse{}, nil
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
