package analytics

import "testing"

func TestMangleEventName(t *testing.T) {
	tests := []struct {
		Name     string
		Expected string
		Error    bool
	}{
		{Name: "support.terms_of_use_&_privacy_policy_pressed", Expected: "support.terms_of_use_-_privacy_policy_pressed"},
		{Name: "install_attribution.Facebook Installs", Expected: "install_attribution.Facebook-Installs"},
		{Name: "&&abc??", Expected: "-abc-"},
		{Name: "", Error: true},
		{Name: "&&&", Error: true},
	}

	for _, tc := range tests {
		name, err := MangleEventName(tc.Name)

		if tc.Error {
			if err == nil {
				t.Errorf(`MangleEventName("%s") = "%s", expected an error`, tc.Name, name)
			}
		} else if err != nil {
			t.Errorf(`MangleEventName("%s") returned error: %s, expected "%s"`, tc.Name, err, tc.Expected)
		} else if name != tc.Expected {
			t.Errorf(`MangleEventName("%s") = "%s", expected "%s"`, tc.Name, name, tc.Expected)
		}
	}
}
