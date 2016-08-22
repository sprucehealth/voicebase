package hint

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Params is an interface that any object that implements the validate method
// conforms to
type Params interface {
	Validate() error
}

// ListParams represents the parameters for querying through a list of paginated resources.
type ListParams struct {
	Items  []*QueryItem
	Sort   *Sort
	Offset uint64
	Limit  uint64
}

// Sort represents the fields to configure a sort by while querying for a list of resources.
type Sort struct {
	By   string
	Desc bool
}

// QueryItem represents an operation on a particular field of the resource while querying
// for a list of resources
type QueryItem struct {
	Field      string
	Operations []*Operation
}

// Operation represents a logical filter to apply when querying for a list of resources.
type Operation struct {
	Operator Operator
	Operand  string
}

type Operator int

const (
	OperatorGreaterThan Operator = 1 << iota
	OperatorGreaterThanEqualTo
	OperatorLessThan
	OperatorLessThanEqualTo
	OperatorEqualTo
)

func (l *ListParams) Encode() (string, error) {
	var buffer bytes.Buffer

	if len(l.Items) > 0 {
		for _, item := range l.Items {
			if buffer.Len() > 0 {
				buffer.WriteByte('&')
			}

			encoded, err := item.Encode()
			if err != nil {
				return "", err
			}
			buffer.WriteString(encoded)
		}
	}

	if l.Sort != nil {
		if buffer.Len() > 0 {
			buffer.WriteByte('&')
		}
		buffer.WriteString(l.Sort.Encode())
	}

	if l.Offset > 0 {
		if buffer.Len() > 0 {
			buffer.WriteByte('&')
		}
		buffer.WriteString("offset=")
		buffer.WriteString(strconv.FormatUint(uint64(l.Offset), 10))
	}

	if l.Limit > 0 {
		if buffer.Len() > 0 {
			buffer.WriteByte('&')
		}
		buffer.WriteString("limit=")
		buffer.WriteString(strconv.FormatUint(uint64(l.Limit), 10))
	}

	return buffer.String(), nil
}

func (q *Sort) Encode() string {
	var buffer bytes.Buffer
	buffer.WriteString("sort")
	buffer.WriteString("=")
	if q.Desc {
		buffer.WriteString("-")
	}
	buffer.WriteString(url.QueryEscape(q.By))
	return buffer.String()
}

func (q *QueryItem) Encode() (string, error) {
	var buffer bytes.Buffer
	buffer.WriteString(url.QueryEscape(q.Field))
	buffer.WriteString("=")

	if len(q.Operations) > 1 && containsEqualToOperator(q.Operations) {
		return "", errors.New("cannot have multiple operations when there is an equal to operator")
	}

	if len(q.Operations) == 1 && q.Operations[0].Operator == OperatorEqualTo {
		buffer.WriteString(q.Operations[0].Encode())
	} else {
		encodedOperations := make([]string, len(q.Operations))
		for i, operation := range q.Operations {
			encodedOperations[i] = operation.Encode()
		}
		buffer.WriteString(url.QueryEscape("{"+strings.Join(encodedOperations, ",")) + "}")
	}

	return buffer.String(), nil
}

func (o *Operation) Encode() string {
	switch o.Operator {
	case OperatorGreaterThan:
		return fmt.Sprintf(`"gt":"%s"`, o.Operand)
	case OperatorLessThan:
		return fmt.Sprintf(`"lt":"%s"`, o.Operand)
	case OperatorGreaterThanEqualTo:
		return fmt.Sprintf(`"gte":"%s"`, o.Operand)
	case OperatorLessThanEqualTo:
		return fmt.Sprintf(`"lte":"%s"`, o.Operand)
	case OperatorEqualTo:
		return o.Operand
	}

	return ""
}

func containsEqualToOperator(operations []*Operation) bool {
	for _, operation := range operations {
		if operation.Operator == OperatorEqualTo {
			return true
		}
	}
	return false
}
