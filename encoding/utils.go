package encoding

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"strconv"
)

// Defining a type of float that holds the precision in the format entered
// versus compacting the result to scientific exponent notation on marshalling the value
// this is useful to send the value as it was entered across the wire so that the client
// can display it as a string without having to worry about the float value, the exact precision, etc.
type HighPrecisionFloat64 float64

// Note that HighPrecisionFloat is always marshalled and unmarshalled as a string
func (h HighPrecisionFloat64) MarshalJSON() ([]byte, error) {
	var marshalledValue []byte
	marshalledValue = append(marshalledValue, '"')
	marshalledValue = strconv.AppendFloat(marshalledValue, float64(h), 'f', -1, 64)
	marshalledValue = append(marshalledValue, '"')
	return marshalledValue, nil
}

func (h *HighPrecisionFloat64) UnmarshalJSON(data []byte) error {
	strData := string(data)
	if len(strData) < 2 || strData == "null" || strData == "" {
		*h = HighPrecisionFloat64(0)
		return nil
	}

	floatValue, err := strconv.ParseFloat(string(strData[1:len(strData)-1]), 64)
	*h = HighPrecisionFloat64(floatValue)
	return err
}

func (h HighPrecisionFloat64) Float64() float64 {
	return float64(h)
}

func (h HighPrecisionFloat64) String() string {
	return strconv.FormatFloat(float64(h), 'f', -1, 64)
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

func (n NullInt64) MarshalJSON() ([]byte, error) {
	if !n.IsValid {
		return []byte(`null`), nil
	}

	return []byte(strconv.FormatInt(n.Int64Value, 10)), nil
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

func NullInt64FromString(intString string) (NullInt64, error) {
	if intString == "" {
		return NullInt64{}, nil
	}
	int64Value, err := strconv.ParseInt(intString, 10, 64)
	if err != nil {
		return NullInt64{}, err
	}

	return NullInt64{
		IsValid:    true,
		Int64Value: int64Value,
	}, nil
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
func (n NullInt64) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if n.IsValid {
		return e.EncodeElement(strconv.FormatInt(n.Int64Value, 10), start)
	}

	return e.EncodeElement(nil, start)
}

func (n NullInt64) Int64() int64 {
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
type ObjectID struct {
	Int64Value int64
	IsValid    bool
}

func NewObjectID(intID int64) ObjectID {
	objectID := ObjectID{
		Int64Value: intID,
		IsValid:    true,
	}
	return objectID
}

func (id *ObjectID) UnmarshalJSON(data []byte) error {
	strData := string(data)
	// only treating the case of an empty string or a null value
	// as value being 0.
	// otherwise relying on integer parser
	if len(strData) < 2 || strData == "null" || strData == `""` {
		*id = ObjectID{
			Int64Value: 0,
			IsValid:    false,
		}
		return nil
	}
	intID, err := strconv.ParseInt(strData[1:len(strData)-1], 10, 64)
	*id = ObjectID{
		Int64Value: intID,
		IsValid:    true,
	}
	return err
}

func (id ObjectID) MarshalJSON() ([]byte, error) {
	// don't marshal anything if value is not valid
	if !id.IsValid {
		return []byte(`null`), nil
	}

	return []byte(fmt.Sprintf(`"%d"`, id.Int64Value)), nil
}

func (id ObjectID) Int64() int64 {
	return id.Int64Value
}

func (id ObjectID) Int64Ptr() *int64 {
	if !id.IsValid {
		return nil
	}
	return &id.Int64Value
}

func (id *ObjectID) Scan(src interface{}) error {
	var nullInt64 sql.NullInt64
	err := nullInt64.Scan(src)
	if err != nil {
		return err
	}

	*id = ObjectID{
		Int64Value: nullInt64.Int64,
		IsValid:    nullInt64.Valid,
	}
	return nil
}

func (id *ObjectID) String() string {
	return strconv.FormatInt(id.Int64Value, 10)
}
