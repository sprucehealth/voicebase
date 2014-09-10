package email

import "regexp"

// Regular expression from WebCore's HTML5 email input: http://goo.gl/7SZbzj
var emailRegexp = regexp.MustCompile("(?i)" + // case insensitive
	"^[a-z0-9!#$%&'*+/=?^_`{|}~.-]+" + // local part
	"@" +
	"[a-z0-9-]+(\\.[a-z0-9-]+)+$") // domain part

// IsValidEmail returns true if the given string is a valid email address.
//
// It uses a simple regular expression to check the address validity.
func IsValidEmail(email string) bool {
	if len(email) > 254 {
		return false
	}
	return emailRegexp.MatchString(email)
}
