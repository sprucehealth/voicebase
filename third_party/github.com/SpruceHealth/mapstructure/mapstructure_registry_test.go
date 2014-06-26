package mapstructure

import (
	"reflect"
	"testing"
)

type Animal interface {
	Speak() string
}

type horse struct {
	Color  string
	Weight string
	Breed  string
}

func (h horse) TypeName() string {
	return "horse"
}

func (h horse) Speak() string {
	return "neigh"
}

type dog struct {
	Breed string
	Color string
}

func (d dog) TypeName() string {
	return "dog"
}

func (d dog) Speak() string {
	return "woof"
}

type barn struct {
	Animals []Animal
	Name    string
}

func (b barn) TypeName() string {
	return "barn"
}

func TestEmbeddedType(t *testing.T) {
	input := map[string]interface{}{
		"animals": []interface{}{
			map[string]interface{}{
				"type":  "dog",
				"breed": "yorkie",
				"color": "brown",
			},
			map[string]interface{}{
				"type":   "horse",
				"color":  "black",
				"weight": "500kg",
			},
		},
	}

	barnA := barn{}

	decodeConfig := &DecoderConfig{}
	registry := NewTypeRegistry()
	registry.MustRegisterType(dog{})
	registry.MustRegisterType(horse{})
	decodeConfig.Result = &barnA
	decodeConfig.Registry = *registry

	d, err := NewDecoder(decodeConfig)
	if err != nil {
		t.Fatalf("error creating new decoder: %s", err)
	}

	err = d.Decode(input)
	if err != nil {
		t.Fatalf("Got error %s", err)
	}

	if len(barnA.Animals) != 2 {
		t.Fatalf("Expected 2 animals in the barn but got %d", len(barnA.Animals))
	}

	if _, ok := barnA.Animals[0].(dog); !ok {
		t.Fatalf("Expected first item to be a dog instead it was a %s", reflect.TypeOf(barnA.Animals[0]))
	}

	if _, ok := barnA.Animals[1].(horse); !ok {
		t.Fatalf("Expected second item to be a horse instead it was a %s", reflect.TypeOf(barnA.Animals[1]))
	}
}

type barnList struct {
	Barns     []barn
	GroupName string
	Count     int
}

func TestMultipleEmbeddedType(t *testing.T) {
	input := map[string]interface{}{
		"count":     2,
		"groupname": "Old Macdonald's Barns",
		"barns": []interface{}{
			map[string]interface{}{
				"type": "barn",
				"animals": []interface{}{
					map[string]interface{}{
						"type":  "dog",
						"breed": "yorkie",
						"color": "brown",
					},
					map[string]interface{}{
						"type":   "horse",
						"color":  "black",
						"weight": "500kg",
					},
				},
			},
			map[string]interface{}{
				"type": "barn",
				"animals": []interface{}{
					map[string]interface{}{
						"type":  "dog",
						"breed": "golden_retriever",
						"color": "brown",
					},
					map[string]interface{}{
						"type":   "dog",
						"color":  "black",
						"weight": "500kg",
					},
				},
			},
		},
	}

	barns := barnList{}

	decodeConfig := &DecoderConfig{}
	registry := NewTypeRegistry()
	registry.MustRegisterType(barn{})
	registry.MustRegisterType(dog{})
	registry.MustRegisterType(horse{})
	decodeConfig.Result = &barns
	decodeConfig.Registry = *registry

	d, err := NewDecoder(decodeConfig)
	if err != nil {
		t.Fatalf("error creating new decoder: %s", err)
	}

	err = d.Decode(input)
	if err != nil {
		t.Fatalf("Got error %s", err)
	}

	if len(barns.Barns) != 2 {
		t.Fatalf("Expected 2 barns instead got %d", len(barns.Barns))
	}

	if barns.Count != 2 {
		t.Fatalf("Expected top level count to be 2 instead got %d", barns.Count)
	}

	if barns.GroupName != input["groupname"] {
		t.Fatalf("Expected groupname to be %s instead it was %s", input["groupname"], barns.GroupName)
	}
}

type notAnimal struct {
	ItemName string
}

func (n notAnimal) TypeName() string {
	return "not_animal"
}

func TestWrongType_Error(t *testing.T) {
	input := map[string]interface{}{
		"animals": []interface{}{
			map[string]interface{}{
				"type":  "dog",
				"breed": "yorkie",
				"color": "brown",
			},
			map[string]interface{}{
				"type":     "not_animal",
				"itemname": "desk",
			},
		},
	}

	barnA := barn{}
	decodeConfig := &DecoderConfig{}
	registry := NewTypeRegistry()
	registry.MustRegisterType(barn{})
	registry.MustRegisterType(dog{})
	registry.MustRegisterType(horse{})
	decodeConfig.Result = &barnA
	decodeConfig.Registry = *registry

	d, err := NewDecoder(decodeConfig)
	if err != nil {
		t.Fatalf("error creating new decoder: %s", err)
	}

	err = d.Decode(input)
	if err == nil {
		t.Fatalf("Expected error but got none", err)
	}
}

func TestUnregisteredType_Error(t *testing.T) {
	input := map[string]interface{}{
		"animals": []interface{}{
			map[string]interface{}{
				"type":  "cow",
				"color": "brown",
			},
			map[string]interface{}{
				"type":   "horse",
				"color":  "black",
				"weight": "500kg",
			},
		},
	}

	barnA := barn{}

	decodeConfig := &DecoderConfig{}
	registry := NewTypeRegistry()

	registry.MustRegisterType(dog{})
	registry.MustRegisterType(horse{})
	decodeConfig.Result = &barnA
	decodeConfig.Registry = *registry

	d, err := NewDecoder(decodeConfig)
	if err != nil {
		t.Fatalf("error creating new decoder: %s", err)
	}

	err = d.Decode(input)
	if err == nil {
		t.Fatalf("Expected error but got none", err)
	}
}

func TestNoType_Error(t *testing.T) {
	input := map[string]interface{}{
		"animals": []interface{}{
			map[string]interface{}{
				"type":  "dog",
				"color": "brown",
			},
			map[string]interface{}{
				"type":   "horse",
				"color":  "black",
				"weight": "500kg",
			},
		},
	}

	barnA := barn{}

	err := Decode(input, &barnA)
	if err == nil {
		t.Fatalf("Expected error but got none", err)
	}
}

func TestAlreadyRegisteredType_Error(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected a panic from the registry for registering same type twice but got no panic")
		}
	}()

	registry := NewTypeRegistry()

	registry.MustRegisterType(dog{})
	registry.MustRegisterType(dog{})
}
