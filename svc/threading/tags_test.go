package threading

import "testing"

func TestValidateTag(t *testing.T) {
	cases := map[string]bool{
		// True
		"a":              true,
		"foo":            true,
		"1":              true,
		"yes_underscore": true,
		"yes-dash":       true,
		"föœ":            true,
		"#foo":           true,
		// False
		"":          false,
		"no spaces": false,
		"🙃":         false, // For now, no emoji
		HiddenTagPrefix + "nothidden": false,
	}
	for tag, valid := range cases {
		t.Run(tag, func(t *testing.T) {
			if v := ValidateTag(tag, false); v != valid {
				t.Errorf("ValidateTag(%q) = %t expected %t", tag, v, valid)
			}
		})
	}

	if v := ValidateTag("$hidden", true); !v {
		t.Errorf("ValidateTag(%q) = %t expected %t", "$hidden", v, true)
	}
}
