package gqldecode

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/sprucehealth/backend/libs/textutil"
)

const tagName = "gql"

type ErrValidationFailed struct {
	Field  string
	Reason string
}

func (e ErrValidationFailed) Error() string {
	return fmt.Sprintf("gqldecode: field %s failed validation: %s", e.Field, e.Reason)
}

func Decode(in map[string]interface{}, out interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				panic(r)
			}
		}
	}()
	outV := reflect.ValueOf(out)
	if outV.Kind() != reflect.Ptr || outV.Type().Elem().Kind() != reflect.Struct {
		return fmt.Errorf("gqldecode: Decode requires a pointer to a struct")
	}
	decodeStruct(in, outV.Elem())
	return nil
}

func decodeStruct(in map[string]interface{}, out reflect.Value) {
	si := infoForStruct(out.Type())
	for name, value := range in {
		fieldInfo := si.fields[name]
		if fieldInfo == nil {
			errf("gqldecode: field %s not found for struct %T", name, out)
		}
		field := out.Field(fieldInfo.index)
		decodeValue(value, field, fieldInfo)
	}
}

func decodeValue(v interface{}, out reflect.Value, fi *structFieldInfo) {
	switch out.Kind() {
	case reflect.String:
		s := v.(string)
		if fi.nonEmpty && s == "" {
			panic(ErrValidationFailed{Field: fi.name, Reason: "value may not be empty"})
		}
		if fi.plane0Unicode && !textutil.IsValidPlane0Unicode(s) {
			panic(ErrValidationFailed{Field: fi.name, Reason: "value must be plane0 unicode"})
		}
		if !utf8.ValidString(s) {
			panic(ErrValidationFailed{Field: fi.name, Reason: "value must be utf8 encoded"})
		}
		out.SetString(s)
	case reflect.Int:
		out.SetInt(int64(v.(int)))
	case reflect.Bool:
		out.SetBool(v.(bool))
	case reflect.Float64:
		out.SetFloat(v.(float64))
	case reflect.Slice:
		inS := v.([]interface{})
		outS := reflect.MakeSlice(out.Type(), len(inS), len(inS))
		for i, v := range inS {
			decodeValue(v, outS.Index(i), fi)
		}
		out.Set(outS)
	case reflect.Struct:
		decodeStruct(v.(map[string]interface{}), out)
	case reflect.Ptr:
		if out.IsNil() {
			out.Set(reflect.New(out.Type().Elem()))
		}
		decodeValue(v, out.Elem(), fi)
	default:
		errf("gqldecode: unknown kind %s", out.Kind())
	}
}

func errf(msg string, v ...interface{}) {
	panic(fmt.Errorf("gqldecode: "+msg, v...))
}

type structFieldInfo struct {
	index         int
	name          string
	nonEmpty      bool
	plane0Unicode bool
}

type structInfo struct {
	fields map[string]*structFieldInfo
}

var (
	structTypeCacheMu sync.RWMutex
	structTypeCache   = make(map[reflect.Type]*structInfo) // struct type -> field name -> field info
)

func infoForStruct(structType reflect.Type) *structInfo {
	structTypeCacheMu.RLock()
	sm := structTypeCache[structType]
	structTypeCacheMu.RUnlock()
	if sm != nil {
		return sm
	}

	structTypeCacheMu.Lock()
	defer structTypeCacheMu.Unlock()

	// Check again in case someone beat us
	sm = structTypeCache[structType]
	if sm != nil {
		return sm
	}

	sm = &structInfo{
		fields: make(map[string]*structFieldInfo),
	}
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}
		tag := field.Tag
		tagValue := tag.Get(tagName)
		tagOptions := strings.Split(tagValue, ",")
		if len(tagOptions) != 0 {
			name := tagOptions[0]
			fi := &structFieldInfo{name: name, index: i}
			for _, opt := range tagOptions[1:] {
				switch opt {
				case "nonempty":
					fi.nonEmpty = true
				case "plane0":
					fi.plane0Unicode = true
				}
			}
			// Check for duplicate field names
			if _, ok := sm.fields[name]; ok {
				errf("duplicate field %s", name)
			}
			sm.fields[name] = fi
		}
	}
	structTypeCache[structType] = sm
	return sm
}
