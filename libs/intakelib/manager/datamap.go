package manager

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func getDataMap(d interface{}) (dataMap, error) {
	switch object := d.(type) {
	case map[string]interface{}:
		return dataMap(object), nil
	case dataMap:
		return object, nil
	}

	return dataMap(nil), fmt.Errorf("Expected map[string]interface but got %T", d)
}

// dataMap provides convenient methods to access data from the
// map of string to empty interface.
type dataMap map[string]interface{}

func (d dataMap) get(key string) interface{} {
	return d[key]
}

func (d dataMap) exists(key string) bool {
	dataVal, ok := d[key]
	return ok && dataVal != nil
}

func (d dataMap) mustGetBool(key string) bool {
	boolVal, ok := d[key]
	if !ok {
		return false
	}

	return boolVal.(bool)
}

func (d dataMap) mustGetString(key string) string {
	strVal, ok := d[key]
	if !ok {
		return ""
	}

	return strVal.(string)
}

func (d dataMap) mustGetFloat32(key string) float32 {
	floatVal, ok := d[key]
	if !ok {
		return 0
	}

	switch f := floatVal.(type) {
	case int:
		return float32(f)
	case float32:
		return f
	case float64:
		return float32(f)
	}

	panic(fmt.Sprintf("Unable to parse %v into float32", floatVal))
}

func (d dataMap) mustGetInt(key string) int {
	intVal, ok := d[key]
	if !ok {
		return 0
	}

	switch i := intVal.(type) {
	case float64:
		return int(i)
	case float32:
		return int(i)
	case int:
		return i
	}

	panic(fmt.Sprintf("Unable to parse %v into int", intVal))
}

func (d dataMap) mustGetInt64(key string) int64 {
	intVal, ok := d[key]
	if !ok {
		return 0
	}

	switch i := intVal.(type) {
	case float64:
		return int64(i)
	case float32:
		return int64(i)
	case int32:
		return int64(i)
	case int:
		return int64(i)
	case string:
		iFromString, err := strconv.ParseInt(i, 10, 64)
		if err != nil {
			panic(err)
		}
		return iFromString
	}

	panic(fmt.Sprintf("Unable to parse %v into int64", intVal))
}

func (d dataMap) getJSONData(key string) ([]byte, error) {
	dataVal, ok := d[key]
	if !ok {
		return nil, nil
	}

	jsonData, err := json.Marshal(dataVal)
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}

func (d dataMap) getInterfaceSlice(key string) ([]interface{}, error) {
	iSliceVal, ok := d[key]
	if !ok {
		return nil, nil
	}

	iSlice, ok := iSliceVal.([]interface{})
	if !ok {
		return nil, fmt.Errorf("Expected %s to be []interface{} but got %T", key, iSliceVal)
	}

	return iSlice, nil
}

func (d dataMap) getStringSlice(key string) ([]string, error) {
	dataVal, ok := d[key]
	if !ok {
		return nil, nil
	}

	switch iSlice := dataVal.(type) {
	case []string:
		return iSlice, nil
	case []interface{}:
		strSlice := make([]string, len(iSlice))
		for i, item := range iSlice {
			strSlice[i], ok = item.(string)
			if !ok {
				return nil, fmt.Errorf("Expected each item for %s to be of type string but got %T", key, item)
			}
		}

		return strSlice, nil
	}

	return nil, fmt.Errorf("Expected type []interface{} or []string for %s got %T", key, dataVal)
}

func (d dataMap) dataMapForKey(key string) (dataMap, error) {
	if !d.exists(key) {
		return nil, nil
	}

	object, ok := d.get(key).(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Expected %s to be of type map[string]interface{} but got %T", key, object)
	}

	return dataMap(object), nil
}

func (d dataMap) requiredKeys(typeName string, keys ...string) error {
	missingKeys := make([]string, 0, len(keys))
	for _, key := range keys {
		if !d.exists(key) {
			missingKeys = append(missingKeys, key)
		}
	}

	if len(missingKeys) > 0 {
		return fmt.Errorf("Following required keys are missing for data type: %s\n%s", typeName, strings.Join(missingKeys, ","))
	}
	return nil
}
