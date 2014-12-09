package encoding

import (
	"encoding/json"
	"testing"
)

type ExampleObject struct {
	ID   ObjectID `json:"testing_id,omitempty"`
	Name string   `json:"name"`
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
 "name": "Hello",
 "testing_id": null
}`
)

func TestObjectIdMarshal(t *testing.T) {
	objId := ObjectID{
		Int64Value: 12345,
		IsValid:    true,
	}

	e1 := &ExampleObject{
		ID:   objId,
		Name: "Hello",
	}

	jsonData, err := json.Marshal(e1)
	if err != nil {
		t.Fatal("Unable to marshal objectId as expected: " + err.Error())
	}

	expectedResult := `{"testing_id":"12345","name":"Hello"}`
	if string(jsonData) != expectedResult {
		t.Fatalf("ObjectId object did not get marshalled as expected. Got %s when expected %s", string(jsonData), expectedResult)
	}

	e2 := &ExampleObject{
		Name: "Hello",
	}

	jsonData, err = json.Marshal(e2)
	if err != nil {
		t.Fatal("Unable to marshal object with no objectId: " + err.Error())
	}

	if string(jsonData) != `{"testing_id":null,"name":"Hello"}` {
		t.Fatalf("ObjectId object did not get marshalled as expected: got %s", string(jsonData))
	}
}

func TestObjectIdUnmarshal(t *testing.T) {
	testObject := &ExampleObject{}
	err := json.Unmarshal([]byte(testObjectString), testObject)
	if err != nil {
		t.Fatal("Unable to unmarshal object as expected")
	}

	if testObject.ID.Int64() != 12345 {
		t.Fatalf("Expected the objectId to be set with 12345. Instead it was set as %d", testObject.ID.Int64())
	}

	testObject = &ExampleObject{}
	err = json.Unmarshal([]byte(testObjectStringWithNull), testObject)

	if err != nil {
		t.Fatal("Unable to unmarshal object as expected: " + err.Error())
	}

	if testObject.ID.IsValid {
		t.Fatalf("Expected the objectId to be set as 0, Instead it was set as %d", testObject.ID.Int64())
	}

	testObject = &ExampleObject{}
	err = json.Unmarshal([]byte(testObjectStringWithEmptyString), testObject)

	if err != nil {
		t.Fatal("Unable to unmarshal object as expected: " + err.Error())
	}

	if testObject.ID.Int64() != 0 {
		t.Fatalf("Expected the objectId to be set as 0, Instead it was set as %d", testObject.ID.Int64())
	}

	testObject = &ExampleObject{}
	err = json.Unmarshal([]byte(testObjectStringWithNoObjectId), testObject)

	if err != nil {
		t.Fatal("Unable to unmarshal object as expected")
	}

	if testObject.ID.IsValid {
		t.Fatal("Expected the objectId to be set as 0")
	}
}
