package deeplink

// https://docs.google.com/document/d/1kuJszqKi45z2WFly0xhWMOLyvw0S5Z7gFAu0K5AgCAk/edit#

import (
	"testing"

	"github.com/sprucehealth/backend/test"
)

const webDomain = "web.domain"

func TestOrgDeeplink(t *testing.T) {
	test.Equals(t, "https://web.domain/org/orgID", OrgURL(webDomain, "orgID"))
}

func TestOrgDetailsDeeplink(t *testing.T) {
	test.Equals(t, "https://web.domain/org/orgID/details", OrgDetailsURL(webDomain, "orgID"))
}

func TestSavedQueryDeeplink(t *testing.T) {
	test.Equals(t, "https://web.domain/org/orgID/list/savedQueryID", SavedQueryURL(webDomain, "orgID", "savedQueryID"))
}

func TestSavedQueryDetailsDeeplink(t *testing.T) {
	test.Equals(t, "https://web.domain/org/orgID/list/savedQueryID/details", SavedQueryDetailsURL(webDomain, "orgID", "savedQueryID"))
}

func TestThreadDeeplink(t *testing.T) {
	test.Equals(t, "https://web.domain/org/orgID/list/savedQueryID/thread/threadID", ThreadURL(webDomain, "orgID", "savedQueryID", "threadID"))
}

func TestThreadShareableDeeplink(t *testing.T) {
	test.Equals(t, "https://web.domain/org/orgID/thread/threadID", ThreadURLShareable(webDomain, "orgID", "threadID"))
}

func TestThreadDetailsDeeplink(t *testing.T) {
	test.Equals(t, "https://web.domain/org/orgID/list/savedQueryID/thread/threadID/details", ThreadDetailsURL(webDomain, "orgID", "savedQueryID", "threadID"))
}

func TestThreadDetailsShareableDeeplink(t *testing.T) {
	test.Equals(t, "https://web.domain/org/orgID/thread/threadID/details", ThreadDetailsURLShareable(webDomain, "orgID", "threadID"))
}

func TestThreadMessageDeeplink(t *testing.T) {
	test.Equals(t, "https://web.domain/org/orgID/list/savedQueryID/thread/threadID/message/threadMessageID", ThreadMessageURL(webDomain, "orgID", "savedQueryID", "threadID", "threadMessageID"))
}

func TestThreadMessageShareableDeeplink(t *testing.T) {
	test.Equals(t, "https://web.domain/org/orgID/thread/threadID/message/threadMessageID", ThreadMessageURLShareable(webDomain, "orgID", "threadID", "threadMessageID"))
}

func TestThreadMessageDetailsDeeplink(t *testing.T) {
	test.Equals(t, "https://web.domain/org/orgID/list/savedQueryID/thread/threadID/message/threadMessageID/details", ThreadMessageDetailsURL(webDomain, "orgID", "savedQueryID", "threadID", "threadMessageID"))
}

func TestOrgSettingsEmailDeeplink(t *testing.T) {
	test.Equals(t, "https://web.domain/org/orgID/settings/email", OrgSettingsEmailURL(webDomain, "orgID"))
}

func TestOrgSettingsPhoneDeeplink(t *testing.T) {
	test.Equals(t, "https://web.domain/org/orgID/settings/phone", OrgSettingsPhoneURL(webDomain, "orgID"))
}

func TestOrgSettingsNotificationsDeeplink(t *testing.T) {
	test.Equals(t, "https://web.domain/org/orgID/settings/notifications", OrgSettingsNotificationsURL(webDomain, "orgID"))
}
