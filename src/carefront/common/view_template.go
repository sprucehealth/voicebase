package common

import (
	"fmt"
	"reflect"
)

type ViewContext map[string]interface{}

func NewViewContext() *ViewContext {
	viewContextMap := make(map[string]interface{})
	viewContext := ViewContext(viewContextMap)
	return &viewContext
}

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

type ViewConditionEvaluationError struct {
	Message string
}

func (v ViewConditionEvaluationError) Error() string {
	return v.Message
}

type ConditionEvaluator interface {
	EvaluateCondition(condition ViewCondition, context ViewContext) (bool, error)
	Operand() string
}

type DataExistsEvaluator int64

func (d DataExistsEvaluator) EvaluateCondition(condition ViewCondition, context ViewContext) (bool, error) {
	if condition.Op != "key_exists" {
		return false, ViewConditionEvaluationError{Message: fmt.Sprintf("Condition evaluation called with wrong operand. Expected key_exists but got %s", condition.Op)}
	}

	_, ok := context.Get(condition.Key)
	return ok, nil
}

func (d DataExistsEvaluator) Operand() string {
	return "key_exists"
}

var conditionEvaluators = make(map[string]ConditionEvaluator)

func init() {
	dataExistsEvaluator := DataExistsEvaluator(0)
	conditionEvaluators[dataExistsEvaluator.Operand()] = dataExistsEvaluator
}

func EvaluateConditionForView(view View, condition ViewCondition, context ViewContext) (bool, error) {
	conditionEvaluator, ok := conditionEvaluators[condition.Op]
	if !ok {
		return false, ViewConditionEvaluationError{Message: fmt.Sprintf("Unable to find condition with op %s for view type %s", condition.Op, view.TypeName())}
	}

	return conditionEvaluator.EvaluateCondition(condition, context)
}
