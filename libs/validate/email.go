package validate

import (
	"regexp"
	"strings"
)

// Regular expression from WebCore's HTML5 email input: http://goo.gl/7SZbzj
var emailRegexp = regexp.MustCompile("(?i)" + // case insensitive
	"^[a-z0-9!#$%&'*+/=?^_`{|}~.-]+" + // local part
	"@" +
	"[a-z0-9-]+(\\.[a-z0-9-]+)+$") // domain part

// Email returns true if the given string is a valid email address.
//
// It uses a simple regular expression to check the address validity.
func Email(email string) bool {
	if len(email) > 254 {
		return false
	}
	if !emailRegexp.MatchString(email) {
		return false
	}
	ix := strings.LastIndex(email, ".")
	if ix < 0 {
		return false
	}
	tld := strings.ToLower(email[ix+1:])
	return TLD(tld)
}
