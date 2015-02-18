package encoding

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	DateSeparator = "-"
	DateFormat    = "YYYY-MM-DD"
)

type Date struct {
	Month int
	Day   int
	Year  int
}

func (d Date) Validate() error {
	if d.Year < 1900 {
		return fmt.Errorf("Invalid year %d in date", d.Year)
	}
	if d.Month < 1 || d.Month > 12 {
		return fmt.Errorf("Invalid month %d in date", d.Month)
	}
	if d.Day < 1 || d.Day > 31 {
		return fmt.Errorf("Invalid day %d in date", d.Day)
	}
	return nil
}

func (d *Date) UnmarshalJSON(data []byte) error {
	strDate := string(data)

	if len(data) < 2 || strDate == "null" || strDate == `""` {
		*d = Date{}
		return nil
	}

	// break up date into components
	dateParts := strings.Split(strDate, DateSeparator)

	if len(dateParts) != 3 {
		return fmt.Errorf("Date incorrectly formatted. Expected format %s", DateFormat)
	}

	if len(dateParts[0]) != 5 || len(dateParts[1]) != 2 || len(dateParts[2]) != 3 {
		return fmt.Errorf("Date incorrectly formatted. Expected format %s", DateFormat)
	}

	dateYear, err := strconv.Atoi(dateParts[0][1:]) // to remove the `"`
	if err != nil {
		return err
	}

	dateMonth, err := strconv.Atoi(dateParts[1])
	if err != nil {
		return err
	}

	dateDay, err := strconv.Atoi(dateParts[2][:len(dateParts[2])-1]) // to remove the `"`
	if err != nil {
		return err
	}

	date := Date{
		Year:  dateYear,
		Month: dateMonth,
		Day:   dateDay,
	}

	if err := date.Validate(); err != nil {
		return err
	}

	*d = date

	return nil
}

func (d Date) MarshalJSON() ([]byte, error) {
	if d.Month == 0 && d.Year == 0 && d.Day == 0 {
		return []byte(`null`), nil
	}

	return []byte(fmt.Sprintf(`"%d-%02d-%02d"`, d.Year, d.Month, d.Day)), nil
}

func (d *Date) Scan(src interface{}) error {
	if src == nil {
		*d = Date{}
		return nil
	}
	switch v := src.(type) {
	case time.Time:
		year, month, day := v.Date()
		*d = Date{Year: year, Month: int(month), Day: day}
		return nil
	case string:
		var err error
		*d, err = ParseDate(v, "YMD", []rune{'/', '-'}, 0)
		return err
	case []byte:
		var err error
		*d, err = ParseDate(string(v), "YMD", []rune{'/', '-'}, 0)
		return err
	}
	return fmt.Errorf("encoding: can't scan type %T into Date", src)
}

func (d *Date) Value() (driver.Value, error) {
	if d == nil {
		return nil, nil
	}
	if d.IsZero() {
		return nil, nil
	}
	return d.ToTime(), nil
}

func (d Date) ToTime() time.Time {
	return time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC)
}

func (d Date) String() string {
	return fmt.Sprintf(`%d-%02d-%02d`, d.Year, d.Month, d.Day)
}

func (d Date) Age() int {
	now := time.Now()
	age := now.Year() - d.Year
	month := int(now.Month())
	if month < d.Month || (month == d.Month && now.Day() < d.Day) {
		age--
	}
	return age
}

func (d Date) IsZero() bool {
	return d.Year == 0 && d.Month == 0 && d.Day == 0
}

func NewDateFromTime(dateTime time.Time) Date {
	dateYear, dateMonth, dateDay := dateTime.Date()
	d := Date{
		Month: int(dateMonth),
		Year:  dateYear,
		Day:   dateDay,
	}
	return d
}

func NewDateFromComponents(dateYear, dateMonth, dateDay string) (Date, error) {
	var d Date
	var err error
	d.Day, err = strconv.Atoi(dateDay)
	if err != nil {
		return d, err
	}

	d.Month, err = strconv.Atoi(dateMonth)
	if err != nil {
		return d, err
	}

	d.Year, err = strconv.Atoi(dateYear)
	if err != nil {
		return d, err
	}

	return d, nil
}

// ParseDate parses a string into year, month, and day providing the
// flexibility of order. The separator is auto-detected and
// can be any rune in the set given. If cutoffYear is given then it
// is used when a two digit year is found. Otherwise, a cutoff of
// the current year is used. Set cutoffYear to less than 0 to prevent
// parsing two digit years and instead return an error.
func ParseDate(dateStr, order string, separators []rune, cutoffYear int) (Date, error) {
	if len(order) != 3 {
		return Date{}, errors.New("encoding.ParseDate: order must be some combination of YMD")
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
		return Date{}, errors.New("encoding.ParseDate: no separator found")
	}
	sepLen := utf8.RuneLen(sep)

	var d Date
	for i := 0; i < 3; i++ {
		vs := dateStr
		if i < 2 {
			idx := strings.IndexRune(dateStr, sep)
			if idx <= 0 {
				return Date{}, fmt.Errorf("encoding.ParseDate: missing part %d", i)
			}
			vs = dateStr[:idx]
			dateStr = dateStr[idx+sepLen:]
		}
		v, err := strconv.Atoi(vs)
		if err != nil {
			return Date{}, fmt.Errorf("encoding.ParseDate: bad number '%s': %s", vs, err.Error())
		}
		switch order[i] {
		case 'Y':
			if len(vs) == 2 {
				if cutoffYear < 0 {
					return Date{}, fmt.Errorf("encoding.ParseDate: two digit year not allowed")
				}
				if x := cutoffYear % 100; v <= x {
					v += cutoffYear - x
				} else {
					v += cutoffYear - x - 100
				}
			}
			d.Year = v
		case 'M':
			d.Month = v
		case 'D':
			d.Day = v
		default:
			return Date{}, fmt.Errorf("encoding.ParseDate: %c not valid in order (must be one of YMD)", order[i])
		}
	}
	if d.Month < 1 || d.Month > 12 {
		return Date{}, fmt.Errorf("encoding.ParseDate: invalid month %d", d.Month)
	}
	if d.Day < 1 || d.Day > 31 {
		return Date{}, fmt.Errorf("encoding.ParseDate: invalid day %d", d.Day)
	}
	return d, nil
}
