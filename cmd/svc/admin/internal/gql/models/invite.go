package models

import (
	"context"

	"sort"

	"github.com/sprucehealth/backend/svc/invite"
)

// PracticeLink represents the an entitie's practice link
type PracticeLink struct {
	OrganizationID string   `json:"organizationID"`
	Token          string   `json:"token"`
	URL            string   `json:"url"`
	Tags           []string `json:"tags"`
}

// TransformPracticeLinksToModel transforms the internal practice links into something unserstood by graphql
func TransformPracticeLinksToModel(ctx context.Context, invs []*invite.OrganizationInvite, inviteAPIDomain string) []*PracticeLink {
	minvs := make([]*PracticeLink, len(invs))
	for i, inv := range invs {
		minvs[i] = TransformPracticeLinkToModel(ctx, inv, inviteAPIDomain)
	}
	return minvs
}

// TransformPracticeLinkToModel transforms the internal practice link into something unserstood by graphql
func TransformPracticeLinkToModel(ctx context.Context, inv *invite.OrganizationInvite, inviteAPIDomain string) *PracticeLink {
	sort.Strings(inv.Tags)
	return &PracticeLink{
		OrganizationID: inv.OrganizationEntityID,
		Token:          inv.Token,
		URL:            invite.OrganizationInviteURL(inviteAPIDomain, inv.Token),
		Tags:           inv.Tags,
	}
}
