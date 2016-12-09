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

func verificationRequirementFromRequest(verificationRequirement invite.InviteVerificationRequirement) (models.VerificationRequirement, error) {
	var vr models.VerificationRequirement
	switch verificationRequirement {
	case invite.VERIFICATION_REQUIREMENT_PHONE_MATCH:
		vr = models.PhoneMatchRequired
	case invite.VERIFICATION_REQUIREMENT_EMAIL:
		vr = models.EmailVerificationRequired
	case invite.VERIFICATION_REQUIREMENT_PHONE:
		vr = models.PhoneVerificationRequired
	default:
		return models.VerificationRequirement(""), errors.Errorf("Unknown verification requirement %s", verificationRequirement)
	}

	return vr, nil
}

func verificationRequirementAsResponse(verificationRequirement models.VerificationRequirement) (invite.InviteVerificationRequirement, error) {
	var vr invite.InviteVerificationRequirement
	switch verificationRequirement {
	case models.PhoneMatchRequired:
		vr = invite.VERIFICATION_REQUIREMENT_PHONE_MATCH
	case models.EmailVerificationRequired:
		vr = invite.VERIFICATION_REQUIREMENT_EMAIL
	case models.PhoneVerificationRequired:
		vr = invite.VERIFICATION_REQUIREMENT_PHONE
	case "":
		vr = invite.VERIFICATION_REQUIREMENT_UNKNOWN
	default:
		return invite.VERIFICATION_REQUIREMENT_UNKNOWN, errors.Errorf("Unknown verification requirement %s", verificationRequirement)
	}
	return vr, nil
}
