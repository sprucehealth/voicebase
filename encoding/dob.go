package encoding

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	DOBSeparator = "-"
	DOBFormat    = "YYYY-MM-DD"
)

type DOB struct {
	Month int
	Day   int
	Year  int
}

func (dob DOB) Validate() error {
	if dob.Year < 1900 {
		return fmt.Errorf("Invalid year %d in date of birth", dob.Year)
	}

	if dob.Month < 1 || dob.Month > 12 {
		return fmt.Errorf("Invalid month %d in date of birth", dob.Month)
	}

	if dob.Day < 1 || dob.Day > 31 {
		return fmt.Errorf("Invalid day %d in date of birth", dob.Day)
	}

	return nil
}

func (dob *DOB) UnmarshalJSON(data []byte) error {
	strDOB := string(data)

	if len(data) < 2 || strDOB == "null" || strDOB == `""` {
		*dob = DOB{}
		return nil
	}

	// break up dob into components
	dobParts := strings.Split(strDOB, DOBSeparator)

	if len(dobParts) != 3 {
		return fmt.Errorf("DOB incorrectly formatted. Expected format %s", DOBFormat)
	}

	if len(dobParts[0]) != 5 || len(dobParts[1]) != 2 || len(dobParts[2]) != 3 {
		return fmt.Errorf("DOB incorrectly formatted. Expected format %s", DOBFormat)
	}

	dobYear, err := strconv.Atoi(dobParts[0][1:]) // to remove the `"`
	if err != nil {
		return err
	}

	dobMonth, err := strconv.Atoi(dobParts[1])
	if err != nil {
		return err
	}

	dobDay, err := strconv.Atoi(dobParts[2][:len(dobParts[2])-1]) // to remove the `"`
	if err != nil {
		return err
	}

	d := DOB{
		Year:  dobYear,
		Month: dobMonth,
		Day:   dobDay,
	}

	if err := d.Validate(); err != nil {
		return err
	}

	*dob = d

	return nil
}

func (dob DOB) MarshalJSON() ([]byte, error) {
	if dob.Month == 0 && dob.Year == 0 && dob.Day == 0 {
		return []byte(`null`), nil
	}

	return []byte(fmt.Sprintf(`"%d-%02d-%02d"`, dob.Year, dob.Month, dob.Day)), nil
}

func (dob DOB) ToTime() time.Time {
	return time.Date(dob.Year, time.Month(dob.Month), dob.Day, 0, 0, 0, 0, time.UTC)
}

func (dob DOB) String() string {
	return fmt.Sprintf(`%d-%02d-%02d`, dob.Year, dob.Month, dob.Day)
}

func NewDOBFromTime(dobTime time.Time) DOB {
	dobYear, dobMonth, dobDay := dobTime.Date()
	dob := DOB{
		Month: int(dobMonth),
		Year:  dobYear,
		Day:   dobDay,
	}
	return dob
}

func NewDOBFromString(dobString string) (DOB, error) {
	var dob DOB
	dobParts := strings.Split(dobString, "-")
	if len(dobParts) != 3 {
		return dob, fmt.Errorf("Required dob format is YYYY-MM-DD")
	}

	return NewDOBFromComponents(dobParts[0], dobParts[1], dobParts[2])
}

func NewDOBFromComponents(dobYear, dobMonth, dobDay string) (DOB, error) {
	var dob DOB
	var err error
	dob.Day, err = strconv.Atoi(dobDay)
	if err != nil {
		return dob, err
	}

	dob.Month, err = strconv.Atoi(dobMonth)
	if err != nil {
		return dob, err
	}

	dob.Year, err = strconv.Atoi(dobYear)
	if err != nil {
		return dob, err
	}

	return dob, nil
}

// ParseDOB parses a string into a DOB struct providing the
// flexibility of order. The separator is auto-detected and
// can be any rune in the set given.
func ParseDOB(dobStr, order string, separators []rune) (DOB, error) {
	if len(order) != 3 {
		return DOB{}, errors.New("encoding.ParseDOB: order must be some combination of YMD")
	}

	var sep rune
	for _, r := range separators {
		if idx := strings.IndexRune(dobStr, r); idx > 0 {
			sep = r
			break
		}
	}
	if sep == 0 {
		return DOB{}, errors.New("encoding.ParseDOB: no separator found")
	}
	sepLen := utf8.RuneLen(sep)

	var dob DOB
	for i := 0; i < 3; i++ {
		vs := dobStr
		if i < 2 {
			idx := strings.IndexRune(dobStr, sep)
			if idx <= 0 {
				return dob, fmt.Errorf("encoding.ParseDOB: missing part %d", i)
			}
			vs = dobStr[:idx]
			dobStr = dobStr[idx+sepLen:]
		}
		v, err := strconv.Atoi(vs)
		if err != nil {
			return dob, fmt.Errorf("encoding.ParseDOB: bad number '%s': %s", vs, err.Error())
		}
		switch order[i] {
		case 'Y':
			if len(vs) == 2 {
				// If given 2 digits then if the lower two digits of
				// the current year are greater than or equal than
				// assume it's in this century. Otherwise, the year
				// must be in the last century. This should be fine
				// for birthdays as it covers anyone from 100 years old
				// to 0.
				curYear := time.Now().Year()
				if x := curYear % 100; v <= x {
					v += curYear - x
				} else {
					v += curYear - x - 100
				}
			}
			dob.Year = v
		case 'M':
			dob.Month = v
		case 'D':
			dob.Day = v
		default:
			return DOB{}, fmt.Errorf("encoding.ParseDOB: %r not valid in order (must be one of YMD)", order[i])
		}
	}
	return dob, dob.Validate()
}
