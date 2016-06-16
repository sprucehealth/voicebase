package app_url

import (
	"encoding/json"
	"testing"
)

type testObject struct {
	ActionURL *SpruceAction `json:"action_url"`
	AssetURL  *SpruceAsset  `json:"image_url"`
}

func TestUnMarshallingSpruceAction(t *testing.T) {
	example := `{
		"action_url" : "spruce:///action/test_this_out?parameter_id=1"
		}`

	t1 := testObject{}
	if err := json.Unmarshal([]byte(example), &t1); err != nil {
		t.Fatalf(err.Error())
	} else if t1.ActionURL.name != "test_this_out" {
		t.Fatalf("Expected %s as action name but got %s", "test_this_out", t1.ActionURL.name)
	} else if t1.ActionURL.params.Get("parameter_id") != "1" {
		t.Fatalf("Expected parameter_id to exist in the params but it doesnt")
	}

	example = `{
		"action_url" : "spruce:///"
	}`

	t1 = testObject{}
	if err := json.Unmarshal([]byte(example), &t1); err != nil {
		t.Fatalf(err.Error())
	} else if t1.ActionURL.name != "" {
		t.Fatalf("Expected empty action name instead got %s", t1.ActionURL.name)
	}

	example = `{
		"action_url" : "spruce:///action/testing_this_out_again"
	}`

	t1 = testObject{}
	if err := json.Unmarshal([]byte(example), &t1); err != nil {
		t.Fatalf(err.Error())
	} else if t1.ActionURL.name != "testing_this_out_again" {
		t.Fatalf("Expected action name %s instead got %s", "testing_this_out_again", t1.ActionURL.name)
	} else if len(t1.ActionURL.params) != 0 {
		t.Fatalf("Expected no params instead got %d", len(t1.ActionURL.params))
	}

	example = `{
		"action_url" : ""
	}`

	t1 = testObject{}
	if err := json.Unmarshal([]byte(example), &t1); err != nil {
		t.Fatalf(err.Error())
	} else if t1.ActionURL.name != "" {
		t.Fatalf("Expected action name %s instead got %s", "testing_this_out_again", t1.ActionURL.name)
	} else if len(t1.ActionURL.params) != 0 {
		t.Fatalf("Expected no params instead got %d", len(t1.ActionURL.params))
	}

	example = `{
		"action_url" : "3ttwgwg3"
	}`

	t1 = testObject{}
	if err := json.Unmarshal([]byte(example), &t1); err != nil {
		t.Fatalf(err.Error())
	} else if t1.ActionURL.name != "" {
		t.Fatalf("Expected action name %s instead got %s", "testing_this_out_again", t1.ActionURL.name)
	} else if len(t1.ActionURL.params) != 0 {
		t.Fatalf("Expected no params instead got %d", len(t1.ActionURL.params))
	}
}

func TestUnMarshallingSpruceAsset(t *testing.T) {
	example := `{
		"image_url" : "spruce:///image/test_this_out?parameter_id=1"
		}`

	t1 := testObject{}
	if err := json.Unmarshal([]byte(example), &t1); err != nil {
		t.Fatalf(err.Error())
	} else if t1.AssetURL.name != "test_this_out" {
		t.Fatalf("Expected %s as action name but got %s", "test_this_out", t1.ActionURL.name)
	}

	example = `{
		"image_url" : "spruce:///image/test_this_out"
		}`

	t1 = testObject{}
	if err := json.Unmarshal([]byte(example), &t1); err != nil {
		t.Fatalf(err.Error())
	} else if t1.AssetURL.name != "test_this_out" {
		t.Fatalf("Expected %s as action name but got %s", "test_this_out", t1.ActionURL.name)
	}
}
