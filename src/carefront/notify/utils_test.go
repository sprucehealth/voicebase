package notify

import (
	"carefront/common"
	"sort"
	"testing"
)

func TestSortingCommunicationPreference(t *testing.T) {

	// simple test
	prefs := []*common.CommunicationPreference{
		&common.CommunicationPreference{
			CommunicationType: common.SMS,
		},
		&common.CommunicationPreference{
			CommunicationType: common.Push,
		},
		&common.CommunicationPreference{
			CommunicationType: common.Email,
		},
	}

	sort.Sort(sort.Reverse(ByCommunicationPreference(prefs)))

	if prefs[0].CommunicationType != common.Push {
		t.Fatalf("Expected first item to be push but it was %s instead", prefs[0].CommunicationType)
	} else if prefs[1].CommunicationType != common.SMS {
		t.Fatalf("Expected first item to be sms but it was %s instead", prefs[1].CommunicationType)
	} else if prefs[2].CommunicationType != common.Email {
		t.Fatalf("Expected first item to be email but it was %s instead", prefs[2].CommunicationType)
	}

	// testing duplicates in array
	prefs = []*common.CommunicationPreference{
		&common.CommunicationPreference{
			CommunicationType: common.SMS,
		},
		&common.CommunicationPreference{
			CommunicationType: common.SMS,
		},
		&common.CommunicationPreference{
			CommunicationType: common.Email,
		},
	}

	sort.Sort(sort.Reverse(ByCommunicationPreference(prefs)))

	if prefs[0].CommunicationType != common.SMS {
		t.Fatalf("Expected first item to be sms but it was %s instead", prefs[0].CommunicationType)
	} else if prefs[1].CommunicationType != common.SMS {
		t.Fatalf("Expected first item to be sms but it was %s instead", prefs[1].CommunicationType)
	} else if prefs[2].CommunicationType != common.Email {
		t.Fatalf("Expected first item to be email but it was %s instead", prefs[2].CommunicationType)
	}

	// testing nil value for array
	prefs = nil
	sort.Sort(sort.Reverse(ByCommunicationPreference(prefs)))
}
