// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package schema

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// NewDecoder returns a new Decoder.
func NewDecoder() *Decoder {
	return &Decoder{cache: newCache()}
}

// Decoder decodes values from a map[string][]string to a struct.
type Decoder struct {
	cache *cache
}

// RegisterConverter registers a converter function for a custom type.
func (d *Decoder) RegisterConverter(value interface{}, converterFunc Converter) {
	d.cache.conv[reflect.TypeOf(value)] = converterFunc
}

// Decode decodes a map[string][]string to a struct.
//
// The first parameter must be a pointer to a struct.
//
// The second parameter is a map, typically url.Values from an HTTP request.
// Keys are "paths" in dotted notation to the struct fields and nested structs.
//
// See the package documentation for a full explanation of the mechanics.
func (d *Decoder) Decode(dst interface{}, src map[string][]string) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return errors.New("schema: interface must be a pointer to struct")
	}
	v = v.Elem()
	t := v.Type()
	errors := MultiError{}
	for path, values := range src {
		if parts, err := d.cache.parsePath(path, t); err == nil {
			if err = d.decode(v, path, parts, values); err != nil {
				errors[path] = err
			}
		} else {
			errors[path] = fmt.Errorf("schema: invalid path %q", path)
		}
	}
	if len(errors) > 0 {
		return errors
	}
	err := d.checkRequiredFieldsAtTopLevel(t, src)
	return err
}

// Note that this only checks that the top level fields have been set. If there are structs or a slice of
// structs involved within the struct, it does not do a deep level check of whether or not the
// required values within each of the structs have been set.
func (d *Decoder) checkRequiredFieldsAtTopLevel(t reflect.Type, src map[string][]string) error {
	var err MissingFieldError
	// go through each field of the struct
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// get the field info for this particular type of field within the struct based
		// on its name or alias
		alias := FieldAlias(field)
		fi := d.cache.m[t].fields[alias]
		if fi.isRequired {
			fieldInStructSet := false
			for key, _ := range src {
				// break up the key to get the first part before the dot
				topLevelField := strings.Split(key, ".")[0]
				fmt.Println(topLevelField)
				if src[topLevelField] != nil && topLevelField == alias {
					fieldInStructSet = true
					break
				}
			}
			if fieldInStructSet == false {
				if err == nil {
					err = make([]string, 0)
				}
				err = append(err, alias)
			}
		}
	}
	if len(err) == 0 {
		return nil
	}
	return err
}

// decode fills a struct field using a parsed path.
func (d *Decoder) decode(v reflect.Value, path string, parts []pathPart,
	values []string) error {
	// Get the field walking the struct fields by index.
	for _, idx := range parts[0].path {
		if v.Type().Kind() == reflect.Ptr {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		v = v.Field(idx)
	}

	// Don't even bother for unexported fields.
	if !v.CanSet() {
		return nil
	}

	// Dereference if needed.
	t := v.Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		if v.IsNil() {
			v.Set(reflect.New(t))
		}
		v = v.Elem()
	}

	// Slice of structs. Let's go recursive.
	if len(parts) > 1 {
		idx := parts[0].index
		if v.IsNil() || v.Len() < idx+1 {
			value := reflect.MakeSlice(t, idx+1, idx+1)
			if v.Len() < idx+1 {
				// Resize it.
				reflect.Copy(value, v)
			}
			v.Set(value)
		}
		return d.decode(v.Index(idx), path, parts[1:], values)
	}

	// Simple case.
	if t.Kind() == reflect.Slice {
		var items []reflect.Value
		elemT := t.Elem()
		isPtrElem := elemT.Kind() == reflect.Ptr
		if isPtrElem {
			elemT = elemT.Elem()
		}
		conv := d.cache.conv[elemT]
		if conv == nil {
			return fmt.Errorf("schema: converter not found for %v", elemT)
		}
		for key, value := range values {
			if value == "" {
				// We are just ignoring empty values for now.
				continue
			} else if item := conv(value); item.IsValid() {
				if isPtrElem {
					ptr := reflect.New(elemT)
					ptr.Elem().Set(item)
					item = ptr
				}
				items = append(items, item)
			} else {
				// If a single value is invalid should we give up
				// or set a zero value?
				return ConversionError{path, key}
			}
		}
		value := reflect.Append(reflect.MakeSlice(t, 0, 0), items...)
		v.Set(value)
	} else {
		if values[0] == "" {
			// We are just ignoring empty values for now.
			return nil
		} else if conv := d.cache.conv[t]; conv != nil {
			if value := conv(values[0]); value.IsValid() {
				v.Set(value)
			} else {
				return ConversionError{path, -1}
			}
		} else {
			return fmt.Errorf("schema: converter not found for %v", t)
		}
	}
	return nil
}

// Errors ---------------------------------------------------------------------

// ConversionError stores information about a failed conversion.
type ConversionError struct {
	Key   string // key from the source map.
	Index int    // index for multi-value fields; -1 for single-value fields.
}

func (e ConversionError) Error() string {
	if e.Index < 0 {
		return fmt.Sprintf("schema: error converting value for %q", e.Key)
	}
	return fmt.Sprintf("schema: error converting value for index %d of %q",
		e.Index, e.Key)
}

type MissingFieldError []string

func (e MissingFieldError) Error() string {
	var eString string

	for i, missingField := range e {
		eString = eString + missingField
		if (i + 1) < len(e) {
			eString = eString + ","
		}
	}

	s := "The following parameters are missing: " + eString
	return s
}

// MultiError stores multiple decoding errors.
//
// Borrowed from the App Engine SDK.
type MultiError map[string]error

func (e MultiError) Error() string {
	s := ""
	for _, err := range e {
		s = err.Error()
		break
	}
	switch len(e) {
	case 0:
		return "(0 errors)"
	case 1:
		return s
	case 2:
		return s + " (and 1 other error)"
	}
	return fmt.Sprintf("%s (and %d other errors)", s, len(e)-1)
}
