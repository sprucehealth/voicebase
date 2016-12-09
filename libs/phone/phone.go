package phone

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"unicode"
)

// RegexpE164 matches an E164 formatted phone number
const RegexpE164 = `^\+[1-9](\d{4,14})$`

// NumberFormat represents the format of the phone number representation
type NumberFormat int

const (
	// E164 is the E164 phone number format
	E164 NumberFormat = iota
	// International indicates to represent the phone number with its international country code
	International
	// National indicates to represent the phone number without the country code
	National
	// Pretty presents the phone number in a human readable format
	Pretty
)

const (
	prefix = "+1"

	// the following numbers map to known "words"
	// as listed here: https://www.twilio.com/help/faq/voice/why-am-i-getting-calls-from-these-strange-numbers
	// we'd like to treat them as valid cases so that we are not rejecting
	// phone numbers for valid situations where the user has their phone number blocked
	// or unavailable.
	NumberRestricted  = "+17378742833"
	NumberBlocked     = "+2562533"
	NumberUnknown     = "+8656696"
	NumberAnonymous   = "+266696687"
	NumberUnavailable = "+86282452253"

	// constants that represent handled cases of a phone number
	restricted  = "RESTRICTED"
	blocked     = "BLOCKED"
	unknown     = "UNKNOWN"
	anonymous   = "ANONYMOUS"
	unavailable = "UNAVAILABLE"
)

var (
	digitsToHandledWordsMap = map[string]string{
		NumberRestricted:  restricted,
		NumberBlocked:     blocked,
		NumberUnknown:     unknown,
		NumberAnonymous:   anonymous,
		NumberUnavailable: unavailable,
	}

	handledWordsToDigitsMap = map[string]string{
		restricted:  NumberRestricted,
		blocked:     NumberBlocked,
		unknown:     NumberUnknown,
		anonymous:   NumberAnonymous,
		unavailable: NumberUnavailable,
	}
)

// Number represents a phone number in E164 form
type Number string

// String implements fmt.Stringer to provider a string representation of number.
func (n Number) String() string {
	return string(n)
}

// IsCallable returns true if the phone number can be called, false if not.
func (n Number) IsCallable() bool {
	switch n.String() {
	case "", NumberRestricted, NumberBlocked, NumberUnknown, NumberUnavailable, NumberAnonymous:
		return false
	}
	return true
}

// MarshalText implements encoding.TextMarshaler
func (n Number) MarshalText() ([]byte, error) {
	return []byte(n), nil
}

// UnmarshalText implements encoding.TextUnmarshaler
func (n *Number) UnmarshalText(text []byte) error {
	number, err := sanitize(string(text))
	if err != nil {
		return err
	}

	*n = Number(number)
	return nil
}

// Scan implements sql.Scanner to assign a value from a database driver.
func (n *Number) Scan(src interface{}) error {
	var pn string
	var err error
	switch v := src.(type) {
	case string:
		pn, err = sanitize(v)
	case []byte:
		pn, err = sanitize(string(v))
	}
	if err != nil {
		return err
	}
	*n = Number(pn)
	return nil
}

// Value implements sql/driver.Valuer to allow a Number to be used in a sql query.
func (n Number) Value() (driver.Value, error) {
	if len(n) == 0 {
		return nil, nil
	}
	return string(n), nil
}

// Format returns a string representation of the number in the specified format.
// If the number cannot be formatted, an error is returned.
// Note that format currently only works for US phone numbers.
func (n Number) Format(format NumberFormat) (string, error) {
	str := n.String()

	if format == E164 {
		return str, nil
	}

	if formattedStr := digitsToHandledWordsMap[str]; formattedStr != "" {
		return formattedStr, nil
	}

	switch format {
	case International:
		return str[:2] + " " + str[2:5] + " " + str[5:8] + " " + str[8:], nil
	case National:
		return str[2:5] + " " + str[5:8] + " " + str[8:], nil
	case Pretty:
		return "(" + str[2:5] + ") " + str[5:8] + "-" + str[8:], nil
	}

	return "", errors.New("Unsupported format")
}

func (n Number) IsEmpty() bool {
	return string(n) == ""
}

// ParseNumber returns a valid Number object if the number is a valid E.164 format
// and errors if not.
func ParseNumber(number string) (Number, error) {
	strippedPhone, err := sanitize(number)
	if err != nil {
		return Number(""), err
	}

	n := Number(strippedPhone)
	return n, nil
}

// Format is a helper method to format a provided phone number in string form
// into the expected format.
func Format(number string, format NumberFormat) (string, error) {
	p, err := ParseNumber(number)
	if err != nil {
		return "", err
	}

	return p.Format(format)
}

func Ptr(pn Number) *Number {
	if pn.IsEmpty() {
		return nil
	}
	return &pn
}

func sanitize(str string) (string, error) {
	if digits := handledWordsToDigitsMap[str]; digits != "" {
		return digits, nil
	}

	strippedPhone := make([]byte, 0, len(str))
	for _, s := range str {
		if unicode.IsDigit(s) {
			strippedPhone = append(strippedPhone, byte(s))
		}
	}

	if handledWord := digitsToHandledWordsMap["+"+string(strippedPhone)]; handledWord != "" {
		return "+" + string(strippedPhone), nil
	}

	startingIdx := 0
	switch {
	case len(strippedPhone) == 10:
	case len(strippedPhone) == 11 && strippedPhone[0] == '1':
		startingIdx = 1
	default:
		return "", fmt.Errorf("%s is not a valid US phone number", str)
	}
	return prefix + string(strippedPhone[startingIdx:]), nil
}
