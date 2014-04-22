package common

import "reflect"

type ViewContext map[string]interface{}

func (c ViewContext) Set(key string, data interface{}) {
	c[key] = data
}

func (c ViewContext) Get(key string) (interface{}, bool) {
	data, ok := c[key]
	return data, ok
}

type TypeRegistry map[string]reflect.Type

func (t TypeRegistry) RegisterType(typ Typed) TypeRegistry {

	t[typ.TypeName()] = reflect.TypeOf(typ)
	return t
}

func (t TypeRegistry) Map() map[string]reflect.Type {
	return (map[string]reflect.Type)(t)
}

type Typed interface {
	TypeName() string
}

type View interface {
	Typed
	Render(context ViewContext) (map[string]interface{}, error)
}

type ViewRenderingError struct {
	Message string
}

func (v ViewRenderingError) Error() string {
	return v.Message
}

func NewViewRenderingError(message string) ViewRenderingError {
	return ViewRenderingError{Message: message}
}

type ViewCondition struct {
	Op  string `json:"op"`
	Key string `json:"key"`
}

type ConditionEvaluator interface {
	EvaluateCondition(condition ViewCondition, context ViewContext) bool
	Operand() string
}

type DataExistsEvaluator int64

func (d DataExistsEvaluator) EvaluateCondition(condition ViewCondition, context ViewContext) bool {
	if condition.Op != "key_exists" {
		return false
	}

	_, ok := context.Get(condition.Key)
	return ok
}

func (d DataExistsEvaluator) Operand() string {
	return "key_exists"
}

var ConditionEvaluators map[string]ConditionEvaluator = make(map[string]ConditionEvaluator)

func init() {
	dataExistsEvaluator := DataExistsEvaluator(0)
	ConditionEvaluators[dataExistsEvaluator.Operand()] = dataExistsEvaluator
}
