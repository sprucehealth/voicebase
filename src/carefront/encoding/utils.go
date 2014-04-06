package encoding

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type NullInt64 struct {
	IsNull     bool
	Int64Value int64
}

func NullInt64FromSql(nullInt64 sql.NullInt64) NullInt64 {
	return NullInt64{
		IsNull:     !nullInt64.Valid,
		Int64Value: nullInt64.Int64,
	}
}

func (n *NullInt64) MarshalJSON() ([]byte, error) {
	if n.IsNull {
		return []byte(`null`), nil
	}

	return []byte(fmt.Sprintf(`%d`, n.Int64Value)), nil
}

func (n *NullInt64) UnmarshalJSON(data []byte) error {
	strData := string(data)

	if strData == "null" {
		*n = NullInt64{
			IsNull: true,
		}
		return nil
	}

	intValue, err := strconv.ParseInt(strData, 10, 64)
	*n = NullInt64{
		IsNull:     false,
		Int64Value: intValue,
	}

	fmt.Printf("%+v", *n)

	return err
}

// need to unmarshal any integer elements that can possibly be returned as nil values
// from dosespot, as indicated by the attribute xsi:nil being set to true.
// I could be doing something incorrectly, but golang seems to not handle
// empty elements for integer types well. Using this custom unmarshaller to
// get around the problem
func (n *NullInt64) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var num int64

	// nothing to do if the value is indicated to be nil via the attribute
	// form of element would be: <elementName xsi:nil="true" />
	if len(start.Attr) > 0 {
		if start.Attr[0].Name.Local == "nil" && start.Attr[0].Value == "true" {
			*n = NullInt64{
				IsNull: true,
			}
			// still decoding to consume the element in the xml document
			d.DecodeElement(&num, &start)
			return nil
		}
	}

	err := d.DecodeElement(&num, &start)
	*n = NullInt64{
		IsNull:     false,
		Int64Value: num,
	}

	return err
}
func (n *NullInt64) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	value := ""
	if n.IsNull {
		if start.Attr == nil {
			start.Attr = make([]xml.Attr, 0)
		}
		start.Attr = append(start.Attr, xml.Attr{
			Name: xml.Name{
				Local: "xsi:nil",
			},
			Value: "true",
		})
	} else {
		value = strconv.FormatInt(n.Int64Value, 10)
	}

	return e.EncodeElement(value, start)
}

func (n *NullInt64) Int64() int64 {
	return n.Int64Value
}

// This is an object used for the (un)marshalling
// of data models ids, such that null values passed from the client
// can be treated as 0 values.
type ObjectId int64

func (id *ObjectId) UnmarshalJSON(data []byte) error {

	strData := string(data)
	// only treating the case of an empty string or a null value
	// as value being 0.
	// otherwise relying on integer parser
	if len(data) < 2 || strData == "null" || strData == `""` {
		*id = 0
		return nil
	}
	intId, err := strconv.ParseInt(strData[1:len(strData)-1], 10, 64)
	*id = ObjectId(intId)
	return err
}

func (id *ObjectId) MarshalJSON() ([]byte, error) {
	if id == nil {
		return []byte(`null`), nil
	}

	return []byte(fmt.Sprintf(`"%d"`, *id)), nil
}

func NewObjectId(intId int64) *ObjectId {
	objectId := ObjectId(intId)
	return &objectId
}

func (id *ObjectId) Int64() int64 {
	if id == nil {
		return 0
	}
	return int64(*id)
}

const (
	DOB_SEPARATOR = "-"
	DOB_FORMAT    = "YYYY-MM-DD"
)

type Dob struct {
	Month int
	Day   int
	Year  int
}

func (dob *Dob) UnmarshalJSON(data []byte) error {
	strDob := string(data)

	if len(data) < 2 || strDob == "null" || strDob == `""` {
		*dob = Dob{}
		return nil
	}

	// break up dob into components (of the format MM/DD/YYYY)
	dobParts := strings.Split(strDob, DOB_SEPARATOR)

	if len(dobParts) < 3 {
		return fmt.Errorf("Dob incorrectly formatted. Expected format %s", DOB_FORMAT)
	}

	if len(dobParts[0]) != 5 || len(dobParts[1]) != 2 || len(dobParts[2]) != 3 {
		return fmt.Errorf("Dob incorrectly formatted. Expected format %s", DOB_FORMAT)
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

	*dob = Dob{
		Year:  dobYear,
		Month: dobMonth,
		Day:   dobDay,
	}

	return nil
}

func (dob *Dob) MarshalJSON() ([]byte, error) {
	if dob == nil {
		return []byte(`null`), nil
	}

	return []byte(fmt.Sprintf(`"%d-%02d-%02d"`, dob.Year, dob.Month, dob.Day)), nil
}

func (dob *Dob) ToTime() time.Time {
	return time.Date(dob.Year, time.Month(dob.Month), dob.Day, 0, 0, 0, 0, time.UTC)
}

func NewDobFromTime(dobTime time.Time) Dob {
	dobYear, dobMonth, dobDay := dobTime.Date()
	dob := Dob{}
	dob.Month = int(dobMonth)
	dob.Year = dobYear
	dob.Day = dobDay
	return dob
}

func NewDobFromComponents(dobYear, dobMonth, dobDay string) (Dob, error) {
	var dob Dob
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
