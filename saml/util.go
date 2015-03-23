package saml

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

var (
	reTagRemove = regexp.MustCompile(`[^\w\s-]`)
	reTagSpaces = regexp.MustCompile(`[-\s]+`)
)

func tagFromText(v string) string {
	v = reTagRemove.ReplaceAllString(v, "")
	v = strings.ToLower(v)
	v = reTagSpaces.ReplaceAllString(v, "_")
	return v
}

func boolPtr(b bool) *bool {
	return &b
}

func validateQuestion(q *Question) error {
	switch q.Details.Type {
	case "q_type_single_select", "q_type_multiple_choice":
		if q.Details.Summary == "" {
			return errors.New("missing summary text")
		}
		if len(q.Details.Answers) == 0 && len(q.Details.AnswerGroups) == 0 {
			return errors.New("missing potential answers")
		}
	case "q_type_free_text":
		if len(q.Details.Answers) != 0 || len(q.Details.AnswerGroups) != 0 {
			return errors.New("free text questions cannot have potential answers")
		}
	}
	return nil
}

func clone(in interface{}) interface{} {
	vin := reflect.ValueOf(in)
	vout := reflect.New(vin.Type())
	cloneValue(vin, vout.Elem())
	return vout.Elem().Interface()
}

func cloneValue(vin, vout reflect.Value) {
	k := vin.Kind()
	switch k {
	default:
		panic(fmt.Sprintf("Unsupported kind %s in clone", k))
	case reflect.Int, reflect.Int64, reflect.Float64, reflect.String, reflect.Bool:
		// Immutable types can be used as is
		vout.Set(vin)
	case reflect.Struct:
		n := vin.NumField()
		for i := 0; i < n; i++ {
			fin := vin.Field(i)
			fout := vout.Field(i)
			cloneValue(fin, fout)
		}
	case reflect.Slice:
		if !vin.IsNil() {
			t := vin.Type()
			sl := reflect.MakeSlice(t, vin.Len(), vin.Len()) // Explicitely set cap to len
			vout.Set(sl)
			switch t.Elem().Kind() {
			default:
				n := vin.Len()
				for i := 0; i < n; i++ {
					cloneValue(vin.Index(i), vout.Index(i))
				}
			case reflect.Int, reflect.Int64, reflect.Float64, reflect.String, reflect.Bool:
				// Immutable types can be optimized by using reflect.Copy
				reflect.Copy(sl, vin)
			}
		}
	case reflect.Ptr:
		if vout.Kind() != reflect.Ptr {
			panic("wtf? no ptr yo")
		}
		if !vin.IsNil() {
			vin = vin.Elem()
			v := reflect.New(vin.Type())
			vout.Set(v)
			cloneValue(vin, v.Elem())
		}
	}
}
