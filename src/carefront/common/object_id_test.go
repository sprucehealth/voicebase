package common

import (
	"encoding/json"
	"testing"
)

type ExampleObject struct {
	Id   *ObjectId `json:"testing_id,omitempty"`
	Name string    `json:"name"`
}

const (
	testObjectString = `{
 "testing_id": "12345",
 "name": "Hello"
}`
	testObjectStringWithEmptyString = `{
 "testing_id": "",
 "name": "Hello"
}`

	testObjectStringWithNull = `{
 "testing_id": null,
 "name": "Hello"
}`

	testObjectStringWithNoObjectId = `{
 "name": "Hello"
}`
)

func TestObjectIdMarshal(t *testing.T) {
	objId := ObjectId(12345)

	e1 := &ExampleObject{
		Id:   &objId,
		Name: "Hello",
	}

	_, err := json.Marshal(e1)
	if err != nil {
		t.Fatal("Unable to marshal objectId as expected: " + err.Error())
	}

	e2 := &ExampleObject{
		Name: "Hello",
	}

	jsonData, err := json.Marshal(e2)
	if err != nil {
		t.Fatal("Unable to marshal object with no objectId: " + err.Error())
	}

	if string(jsonData) != `{"name":"Hello"}` {
		t.Fatal("ObjectId object did not get marshalled as expected")
	}
}

func TestObjectIdUnmarshal(t *testing.T) {
	testObject := &ExampleObject{}
	err := json.Unmarshal([]byte(testObjectString), testObject)
	if err != nil {
		t.Fatal("Unable to unmarshal object as expected")
	}

	if testObject.Id.Int64() != 12345 {
		t.Fatalf("Expected the objectId to be set with 12345. Instead it was set as %d", testObject.Id.Int64())
	}

	testObject = &ExampleObject{}
	err = json.Unmarshal([]byte(testObjectStringWithNull), testObject)

	if err != nil {
		t.Fatal("Unable to unmarshal object as expected: " + err.Error())
	}

	if testObject.Id != nil {
		t.Fatalf("Expected the objectId to be set as 0, Instead it was set as %d", testObject.Id.Int64())
	}

	testObject = &ExampleObject{}
	err = json.Unmarshal([]byte(testObjectStringWithEmptyString), testObject)

	if err != nil {
		t.Fatal("Unable to unmarshal object as expected: " + err.Error())
	}

	if testObject.Id.Int64() != 0 {
		t.Fatalf("Expected the objectId to be set as 0, Instead it was set as %d", testObject.Id.Int64())
	}

	testObject = &ExampleObject{}
	err = json.Unmarshal([]byte(testObjectStringWithNoObjectId), testObject)

	if err != nil {
		t.Fatal("Unable to unmarshal object as expected")
	}

	if testObject.Id != nil {
		t.Fatal("Expected the objectId to be set as 0")
	}
}
