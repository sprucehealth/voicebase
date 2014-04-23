package encoding

import (
	"bytes"
	"database/sql"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Defining a type of float that holds the precision in the format entered
// versus compacting the result to scientific exponent notation on marshalling the value
// this is useful to send the value as it was entered across the wire so that the client
// can display it as a string without having to worry about the float value, the exact precision, etc.
type HighPrecisionFloat64 float64

func (h *HighPrecisionFloat64) MarshalJSON() ([]byte, error) {
	var marshalledValue bytes.Buffer
	marshalledValue.WriteString("\"")
	marshalledValue.WriteString(strconv.FormatFloat(float64(*h), 'f', -1, 64))
	marshalledValue.WriteString("\"")
	return marshalledValue.Bytes(), nil
}

func (h *HighPrecisionFloat64) UnmarshalJSON(data []byte) error {
	strData := string(data)
	if strData == "null" || strData == "" {
		*h = HighPrecisionFloat64(0)
		return nil
	}

	floatValue, err := strconv.ParseFloat(string(strData[1:len(strData)-1]), 64)
	*h = HighPrecisionFloat64(floatValue)
	return err
}

func (h *HighPrecisionFloat64) Float64() float64 {
	return float64(*h)
}

func (h *HighPrecisionFloat64) String() string {
	return strconv.FormatFloat(float64(*h), 'f', -1, 64)
}

func (h *HighPrecisionFloat64) Scan(src interface{}) error {
	var nullFloat64 sql.NullFloat64
	err := nullFloat64.Scan(src)
	if err != nil {
		return err
	}

	*h = HighPrecisionFloat64(nullFloat64.Float64)
	return nil
}

type NullInt64 struct {
	IsValid    bool
	Int64Value int64
}

func (n *NullInt64) MarshalJSON() ([]byte, error) {
	if !n.IsValid {
		return []byte(`null`), nil
	}

	return []byte(fmt.Sprintf(`%d`, n.Int64Value)), nil
}

func (n *NullInt64) UnmarshalJSON(data []byte) error {
	strData := string(data)

	if strData == "null" {
		*n = NullInt64{}
		return nil
	}

	intValue, err := strconv.ParseInt(strData, 10, 64)
	*n = NullInt64{
		IsValid:    true,
		Int64Value: intValue,
	}

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
			*n = NullInt64{}
			// still decoding to consume the element in the xml document
			d.DecodeElement(&num, &start)
			return nil
		}
	}

	err := d.DecodeElement(&num, &start)
	*n = NullInt64{
		IsValid:    true,
		Int64Value: num,
	}

	return err
}
func (n *NullInt64) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if n.IsValid {
		return e.EncodeElement(strconv.FormatInt(n.Int64Value, 10), start)
	}

	return e.EncodeElement(nil, start)
}

func (n *NullInt64) Int64() int64 {
	return n.Int64Value
}

func (n *NullInt64) Scan(src interface{}) error {
	var nullInt sql.NullInt64
	err := nullInt.Scan(src)
	if err != nil {
		return err
	}

	n.IsValid = nullInt.Valid
	n.Int64Value = nullInt.Int64
	return nil
}

// This is an object used for the (un)marshalling
// of data models ids, such that null values passed from the client
// can be treated as 0 values.
type ObjectId struct {
	Int64Value int64
	IsValid    bool
}

func (id *ObjectId) UnmarshalJSON(data []byte) error {

	strData := string(data)
	// only treating the case of an empty string or a null value
	// as value being 0.
	// otherwise relying on integer parser
	if len(data) < 2 || strData == "null" || strData == `""` {
		*id = ObjectId{
			Int64Value: 0,
			IsValid:    false,
		}
		return nil
	}
	intId, err := strconv.ParseInt(strData[1:len(strData)-1], 10, 64)
	*id = ObjectId{
		Int64Value: intId,
		IsValid:    true,
	}
	return err
}

func (id *ObjectId) MarshalJSON() ([]byte, error) {
	// don't marshal anything if value is not valid
	if !id.IsValid {
		return []byte(`null`), nil
	}

	return []byte(fmt.Sprintf(`"%d"`, id.Int64Value)), nil
}

func NewObjectId(intId int64) ObjectId {
	objectId := ObjectId{
		Int64Value: intId,
		IsValid:    true,
	}
	return objectId
}

func (id *ObjectId) Int64() int64 {
	return id.Int64Value
}

func (id *ObjectId) Scan(src interface{}) error {
	var nullInt64 sql.NullInt64
	err := nullInt64.Scan(src)
	if err != nil {
		return err
	}

	*id = ObjectId{
		Int64Value: nullInt64.Int64,
		IsValid:    nullInt64.Valid,
	}
	return nil
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

func (dob *Dob) String() string {
	return fmt.Sprintf(`%d-%02d-%02d`, dob.Year, dob.Month, dob.Day)
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
