package manager

import "testing"

type mockDataSource_pharmacyScreen struct {
	value bool
	questionAnswerDataSource
}

func (m *mockDataSource_pharmacyScreen) valueForKey(key string) interface{} {
	return m.value
}

func TestPharmacyScreen_requirementsMet(t *testing.T) {
	s := &pharmacyScreen{}
	m := &mockDataSource_pharmacyScreen{}

	// if pharmacy has not been set then requirements should not be met
	if res, err := s.requirementsMet(m); err == nil || res {
		t.Fatal("Expected requirements to not be met when pharmacy is not set")
	}

	// requirements not met if value for pharmacy being set is false
	m.value = false
	if res, err := s.requirementsMet(m); err == nil || res {
		t.Fatal("Expected requirements to not be met when pharmacy is not set")
	}

	m.value = true
	if res, err := s.requirementsMet(m); err != nil {
		t.Fatal(err)
	} else if !res {
		t.Fatal("Expected pharmacy screen to have its requirements met")
	}
}
