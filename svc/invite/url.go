package invite

// OrganizationInviteURL returns the url for organization invites
func OrganizationInviteURL(inviteAPIDomain, inviteToken string) string {
	return inviteAPIDomain + "/" + inviteToken
}
