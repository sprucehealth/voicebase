package phone

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"regexp"
)

var (
	matcher = regexp.MustCompile(`^\+[1-9](\d{4,14})$`)
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
	*n = Number(string(text))
	return n.Validate()
}

// Scan implements sql.Scanner to assign a value from a database driver.
func (n *Number) Scan(src interface{}) error {
	switch v := src.(type) {
	case string:
		*n = Number(v)
	case []byte:
		*n = Number(string(v))
	}
	return n.Validate()
}

// Value implements sql/driver.Valuer to allow a Number to be used in a sql query.
func (n Number) Value() (driver.Value, error) {
	if len(n) == 0 {
		return nil, nil
	}
	return string(n), nil
}

// Validate ensures that phone number is a valid number in E164 format.
func (n Number) Validate() error {
	if !matcher.Match([]byte(n)) {
		return fmt.Errorf("%s does not conform to E.164 phone number format", n)
	}
	return nil
}

// Format returns a string representation of the number in the specified format.
// If the number cannot be formatted, an error is returned.
// Note that format currently only works for US phone numbers.
func (n Number) Format(format NumberFormat) (string, error) {
	str := n.String()

	if !(str[:2] == "+1" && len(str) == 12) {
		return "", errors.New("Format only supported for US phone numbers")
	}

	switch format {
	case E164:
		return str, nil
	case International:
		return str[:2] + " " + str[2:5] + " " + str[5:8] + " " + str[8:], nil
	case National:
		return str[2:5] + " " + str[5:8] + " " + str[8:], nil
	}
	return "", errors.New("Unsupported format")
}

// ParseNumber returns a valid Number object if the number is a valid E.164 format
// and errors if not.
func ParseNumber(number string) (Number, error) {
	n := Number(number)
	if err := n.Validate(); err != nil {
		return Number(""), err
	}
	return n, nil
}
