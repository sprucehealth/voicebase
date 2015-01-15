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

func (dob DOB) Age() int {
	now := time.Now()
	age := now.Year() - dob.Year
	month := int(now.Month())
	if month < dob.Month || (month == dob.Month && now.Day() < dob.Day) {
		age--
	}
	return age
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
	year, month, day, err := ParseDate(dobStr, order, separators, 0)
	if err != nil {
		return DOB{}, err
	}
	dob := DOB{
		Year:  year,
		Month: month,
		Day:   day,
	}
	return dob, dob.Validate()
}

// ParseDate parses a string into year, month, and day providing the
// flexibility of order. The separator is auto-detected and
// can be any rune in the set given. If cutoffYear is given then it
// is used when a two digit year is found. Otherwise, a cutoff of
// the current year is used. Set cutoffYear to less than 0 to prevent
// parsing two digit years and instead return an error.
func ParseDate(dateStr, order string, separators []rune, cutoffYear int) (year, month, day int, err error) {
	if len(order) != 3 {
		return 0, 0, 0, errors.New("encoding.ParseDate: order must be some combination of YMD")
	}

	if cutoffYear == 0 {
		cutoffYear = time.Now().UTC().Year()
	}

	var sep rune
	for _, r := range separators {
		if idx := strings.IndexRune(dateStr, r); idx > 0 {
			sep = r
			break
		}
	}
	if sep == 0 {
		return 0, 0, 0, errors.New("encoding.ParseDate: no separator found")
	}
	sepLen := utf8.RuneLen(sep)

	for i := 0; i < 3; i++ {
		vs := dateStr
		if i < 2 {
			idx := strings.IndexRune(dateStr, sep)
			if idx <= 0 {
				return 0, 0, 0, fmt.Errorf("encoding.ParseDate: missing part %d", i)
			}
			vs = dateStr[:idx]
			dateStr = dateStr[idx+sepLen:]
		}
		v, err := strconv.Atoi(vs)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("encoding.ParseDate: bad number '%s': %s", vs, err.Error())
		}
		switch order[i] {
		case 'Y':
			if len(vs) == 2 {
				if cutoffYear < 0 {
					return 0, 0, 0, fmt.Errorf("encoding.ParseDate: two digit year not allowed")
				}
				if x := cutoffYear % 100; v <= x {
					v += cutoffYear - x
				} else {
					v += cutoffYear - x - 100
				}
			}
			year = v
		case 'M':
			month = v
		case 'D':
			day = v
		default:
			return 0, 0, 0, fmt.Errorf("encoding.ParseDate: %c not valid in order (must be one of YMD)", order[i])
		}
	}
	if month < 1 || month > 12 {
		return 0, 0, 0, fmt.Errorf("encoding.ParseDate: invalid month %d", month)
	}
	if day < 1 || day > 31 {
		return 0, 0, 0, fmt.Errorf("encoding.ParseDate: invalid day %d", day)
	}
	return year, month, day, nil
}

func ParseDateToTime(dateStr, order string, separators []rune, cutoffYear int) (time.Time, error) {
	year, month, day, err := ParseDate(dateStr, order, separators, cutoffYear)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}
