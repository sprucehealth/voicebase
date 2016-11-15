package server

import (
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/invite"
)

func organizationInviteAsResponse(inv *models.Invite) (*invite.OrganizationInvite, error) {
	if inv.Type != models.OrganizationCodeInvite {
		return nil, errors.Errorf("%+v is not an organization invite", inv)
	}
	return &invite.OrganizationInvite{
		OrganizationEntityID: inv.OrganizationEntityID,
		Token:                inv.Token,
		Tags:                 inv.Tags,
	}, nil
}
