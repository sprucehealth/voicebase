package email

var Types = map[string]*Type{}

type Type struct {
	Key         string      `json:"key"`
	Name        string      `json:"name"`
	TestContext interface{} `json:"test_context"`
}

func MustRegisterType(t *Type) {
	if t.Key == "" {
		panic("Key is empty")
	}
	if t.Name == "" {
		panic("Name is empty")
	}
	if t.TestContext == nil {
		panic("TestContext is nil")
	}
	if Types[t.Key] != nil {
		panic("Key " + t.Key + " is already registered")
	}
	Types[t.Key] = t
}
