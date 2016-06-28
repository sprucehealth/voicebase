package manager

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
)

const (
	delimiter = '|'
)

var (
	questionDescriptor    = "qu"
	screenDescriptor      = "sc"
	sectionDescriptor     = "se"
	subquestionDescriptor = "te"
	units                 = []string{sectionDescriptor, screenDescriptor, questionDescriptor, subquestionDescriptor}
	h                     = fnv.New32a()
)

// layoutUnitID represents an identifier for a layoutUnit in a visitLayout object.
// use newLayoutUnitID() to instantiate an id to its zero value as -1 is considered
// the 0 value. This is done so as to prevent the need for pointers to indicate
// values being set so as to prevent allocations.
// the ID is of the format: se:<section_id>|sc:<screen_index>|qu:<question_index>|te:<text>
type layoutUnitID struct {
	sectionIndex    int
	screenIndex     int
	questionIndex   int
	subquestionInfo string
}

func newLayoutUnitID() *layoutUnitID {
	lUnitID := &layoutUnitID{
		sectionIndex:  -1,
		screenIndex:   -1,
		questionIndex: -1,
	}
	return lUnitID
}

func (l *layoutUnitID) String() string {
	byteSlice := make([]byte, 0, 4*5) // (number of characters to represent each component) * (number of components)
	var useDelimiter bool

	if l.sectionIndex > 0 {
		byteSlice = append(byteSlice, []byte("se:")...)
		byteSlice = strconv.AppendInt(byteSlice, int64(l.sectionIndex), 10)
		useDelimiter = true
	}

	if l.screenIndex > 0 {
		if useDelimiter {
			byteSlice = append(byteSlice, delimiter)
		}
		byteSlice = append(byteSlice, []byte("sc:")...)
		byteSlice = strconv.AppendInt(byteSlice, int64(l.screenIndex), 10)
		useDelimiter = true
	}

	if l.questionIndex > 0 {
		if useDelimiter {
			byteSlice = append(byteSlice, delimiter)
		}
		byteSlice = append(byteSlice, []byte("qu:")...)
		byteSlice = strconv.AppendInt(byteSlice, int64(l.questionIndex), 10)
		useDelimiter = true
	}

	if l.subquestionInfo != "" {
		if useDelimiter {
			byteSlice = append(byteSlice, delimiter)
		}
		byteSlice = append(byteSlice, []byte("te:")...)
		byteSlice = append(byteSlice, []byte(l.subquestionInfo)...)
	}

	return string(byteSlice)
}

func parseLayoutUnitID(str string) (*layoutUnitID, error) {
	if len(str) == 0 {
		return nil, generateInvalidIDError(str)
	}

	l := &layoutUnitID{}

	var i int
	originalStr := str
	for i < 4 && len(str) > 0 {
		delimiterIndex := strings.IndexRune(str, delimiter)
		if delimiterIndex == -1 {
			delimiterIndex = len(str)
		}

		if len(str) < 4 {
			return nil, generateInvalidIDError(originalStr)
		}

		if str[:2] != units[i] {
			return nil, generateInvalidIDError(originalStr)
		}

		if str[2] != ':' {
			return nil, generateInvalidIDError(originalStr)
		}

		if i < 3 {
			unitIdx, err := strconv.ParseInt(str[3:delimiterIndex], 10, 64)
			if err != nil {
				return nil, err
			}

			switch i {
			case 0:
				l.sectionIndex = int(unitIdx)
			case 1:
				l.screenIndex = int(unitIdx)
			case 2:
				l.questionIndex = int(unitIdx)
			}
		} else {
			l.subquestionInfo = str[3:]
		}

		if delimiterIndex+1 < len(str) {
			str = str[delimiterIndex+1:]
		} else {
			str = str[:0]
		}

		i++
	}

	return l, nil
}

func generateInvalidIDError(str string) error {
	return fmt.Errorf("Invalid layoutUnitID: %s. Valid ID: se:N|sc:N|qu:N|te:<text>", str)
}
