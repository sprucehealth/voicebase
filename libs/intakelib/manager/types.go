package manager

import (
	"fmt"
	"reflect"
)

// typeRegistries stores the mapping of type name -> type of object
// to instantiate
var typeRegistries = map[string]map[string]reflect.Type{
	"screen":    map[string]reflect.Type{},
	"question":  map[string]reflect.Type{},
	"condition": map[string]reflect.Type{},
	"answer":    map[string]reflect.Type{},
}

func mustRegisterScreen(typeName string, s screen) {
	if typeName == "" {
		panic("typeName not set")
	}

	_, ok := typeRegistries["screen"][typeName]
	if ok {
		panic(fmt.Sprintf("%s already defined in registry", typeName))
	}

	typeRegistries["screen"][typeName] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(s)).Interface())
}

func mustRegisterQuestion(typeName string, q question) {
	if typeName == "" {
		panic("typeName not set")
	}

	_, ok := typeRegistries["question"][typeName]
	if ok {
		panic(fmt.Sprintf("%s already defined in registry", typeName))
	}
	typeRegistries["question"][typeName] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(q)).Interface())
}

func mustRegisterAnswer(typeName string, pa patientAnswer) {
	if typeName == "" {
		panic("typeName not set")
	}

	_, ok := typeRegistries["answer"][typeName]
	if ok {
		panic(fmt.Sprintf("%s already defined in registry", typeName))
	}
	typeRegistries["answer"][typeName] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(pa)).Interface())
}

func mustRegisterCondition(typeName string, c condition) {
	if typeName == "" {
		panic("typeName not set")
	}

	_, ok := typeRegistries["condition"][typeName]
	if ok {
		panic(fmt.Sprintf("%s already defined in registry", typeName))
	}

	typeRegistries["condition"][typeName] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(c)).Interface())
}

// questionForType returns an instance of a question object
// of a concrete type based on the typeName.
func questionForType(typeName string) (question, error) {
	dataType, ok := typeRegistries["question"][typeName]
	if !ok {
		return nil, fmt.Errorf("Unable to get populator for type: %s", typeName)
	}

	return reflect.New(dataType).Interface().(question), nil
}

func patientAnswerForType(typeName string) (patientAnswer, error) {
	dataType, ok := typeRegistries["answer"][typeName]
	if !ok {
		return nil, fmt.Errorf("Unable to get populator for type: %s", typeName)
	}

	return reflect.New(dataType).Interface().(patientAnswer), nil
}

// conditionForType returns an instance of a condition object
// of a concrete type based on the typeName.
func conditionForType(typeName string) (condition, error) {
	dataType, ok := typeRegistries["condition"][typeName]
	if !ok {
		return nil, fmt.Errorf("Unable to get populator for type: %s", typeName)
	}

	return reflect.New(dataType).Interface().(condition), nil
}

// screenForType returns an instance of a screen object
// of a concrete type based on the typeName.
func screenForType(typeName string) (screen, error) {
	dataType, ok := typeRegistries["screen"][typeName]
	if !ok {
		return nil, fmt.Errorf("Unable to get populator for type: %s", typeName)
	}

	return reflect.New(dataType).Interface().(screen), nil
}

// getQuestion returns a question object of a concrete type with information populated
// from the map, based on the type assumed to exist in the map.
func getQuestion(d map[string]interface{}, parent layoutUnit, dataSource questionAnswerDataSource) (question, error) {
	questionType, ok := d["type"]
	if !ok {
		return nil, fmt.Errorf("Unable to determine type for question: %v", d)
	}

	q, err := questionForType(questionType.(string))
	if err != nil {
		return nil, err
	}

	return q, q.unmarshalMapFromClient(d, parent, dataSource)
}

// getPatientAnswer returns a patient answer object of a concrete type with information
// populated from the map, based on the type assumed to exist in the map.
func getPatientAnswer(d dataMap) (patientAnswer, error) {
	answerType := d.get("type")
	if answerType == nil {
		return nil, fmt.Errorf("Unable to determine type for patient_answer: %v", d)
	}

	pa, err := patientAnswerForType(answerType.(string))
	if err != nil {
		return nil, err
	}

	return pa, pa.unmarshalMapFromClient(d)
}

// getCondition returns a condition object of a concrete type with information populated
// from the map, based on the type assumed to exist in the map.
func getCondition(d map[string]interface{}) (condition, error) {
	conditionType, ok := d["type"]
	if !ok {
		return nil, fmt.Errorf("Unable to determine type for condition: %v", d)
	}

	c, err := conditionForType(conditionType.(string))
	if err != nil {
		return nil, err
	}

	return c, c.unmarshalMapFromClient(d)
}

// getScreen returns a screen object of a concrete type with information populated
// from the map, based on the type assumed to exist in the map.
func getScreen(d map[string]interface{}, parent layoutUnit, dataSource questionAnswerDataSource) (screen, error) {
	screenType, ok := d["type"]
	if !ok {
		return nil, fmt.Errorf("Unable to determine type for screen: %v", d)
	}

	s, err := screenForType(screenType.(string))
	if err != nil {
		return nil, err
	}

	return s, s.unmarshalMapFromClient(d, parent, dataSource)
}
