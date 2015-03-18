package common

import "testing"

func TestKeyExistsEvaluator(t *testing.T) {
	ctxt := &ViewContext{
		context: map[string]interface{}{
			"testme": true,
		},
	}

	c := ViewCondition{
		Op:  "key_exists",
		Key: "testme",
	}

	eval := KeyExistsEvaluator(0)
	res, err := eval.EvaluateCondition(c, ctxt)
	if err != nil {
		t.Fatalf("Err: %s", err)
	} else if !res {
		t.Fatalf("Expected condition to evaluate to true but it didnt")
	}

	c = ViewCondition{
		Op:  "key_exists",
		Key: "fail",
	}

	res, err = eval.EvaluateCondition(c, ctxt)
	if err != nil {
		t.Fatalf("Err: %s", err)
	} else if res {
		t.Fatalf("Expected condition to evaluate to false but it didnt")
	}

}

func TestAnyKeyExistsEvaluator(t *testing.T) {
	ctxt := &ViewContext{
		context: map[string]interface{}{
			"testme":  true,
			"testme1": true,
		},
	}

	c := ViewCondition{
		Op:   "any_key_exists",
		Keys: []string{"testme", "testme3"},
	}

	eval := AnyKeyExistsEvaluator(0)
	res, err := eval.EvaluateCondition(c, ctxt)
	if err != nil {
		t.Fatalf("Err: %s", err)
	} else if !res {
		t.Fatalf("Expected condition to evaluate to true but it didnt")
	}

	c = ViewCondition{
		Op:   "any_key_exists",
		Keys: []string{"testm2", "testme4"},
	}

	res, err = eval.EvaluateCondition(c, ctxt)
	if err != nil {
		t.Fatalf("Err: %s", err)
	} else if res {
		t.Fatalf("Expected condition to evaluate to false but it didnt")
	}
}
