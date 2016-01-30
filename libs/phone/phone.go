package phone

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"regexp"
	"unicode"
)

var (
	matcher          = regexp.MustCompile(`^\+[1-9](\d{4,14})$`)
	nonDigitsMatcher = regexp.MustCompile(`[^0-9]`)
)

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
)

// Number represents a phone number in E164 form
type Number string

// String implements fmt.Stringer to provider a string representation of number.
func (n Number) String() string {
	return string(n)
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

	switch format {
	case E164:
		return str, nil
	case International:
		return str[:2] + " " + str[2:5] + " " + str[5:8] + " " + str[8:], nil
	case National:
		return str[2:5] + " " + str[5:8] + " " + str[8:], nil
	case Pretty:
		return "(" + str[2:5] + ") " + str[5:8] + "-" + str[8:], nil
	}
	return "", errors.New("Unsupported format")
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

func sanitize(str string) (string, error) {

	strippedPhone := make([]byte, 0, len(str))
	for _, s := range str {
		if unicode.IsDigit(s) {
			strippedPhone = append(strippedPhone, byte(s))
		}
	}

	// strippedPhone := nonDigitsMatcher.ReplaceAllString(str, "")
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
