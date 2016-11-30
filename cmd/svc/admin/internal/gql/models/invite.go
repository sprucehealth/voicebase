package models

import (
	"context"

	"github.com/sprucehealth/backend/svc/invite"
)

type PracticeLink struct {
	OrganizationID string   `json:"organizationID"`
	Token          string   `json:"token"`
	URL            string   `json:"url"`
	Tags           []string `json:"tags"`
}

func TransformPracticeLinksToModel(ctx context.Context, invs []*invite.OrganizationInvite, inviteAPIDomain string) []*PracticeLink {
	minvs := make([]*PracticeLink, len(invs))
	for i, inv := range invs {
		minvs[i] = TransformPracticeLinkToModel(ctx, inv, inviteAPIDomain)
	}
	return minvs
}

func TransformPracticeLinkToModel(ctx context.Context, inv *invite.OrganizationInvite, inviteAPIDomain string) *PracticeLink {
	return &PracticeLink{
		OrganizationID: inv.OrganizationEntityID,
		Token:          inv.Token,
		URL:            invite.OrganizationInviteURL(inviteAPIDomain, inv.Token),
		Tags:           inv.Tags,
	}
}
