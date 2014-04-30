package common

import "fmt"

// The ViewContext is a generic container for data that Views consume
// from when being rendered. Data exists in the ViewContext based on keys
// which are specified in the "content_config" of a view definition in the template
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

func (c ViewContext) Delete(key string) {
	delete(c, key)
}

type View interface {
	Typed
	Render(context ViewContext) (map[string]interface{}, error)
}

type ViewRenderingError struct {
	Message          string
	IsContentMissing bool
}

func (v ViewRenderingError) Error() string {
	return v.Message
}

func NewViewRenderingError(message string) ViewRenderingError {
	return ViewRenderingError{Message: message}
}

// The ViewCondition is a structure found in the ContentConfig
// of views. The Operand defines the type of implementation to use
// to evaluate the condition based on the key, to either true or false
// which essentially indicates whether or not to include the view in the rendering
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

// The ConditionEvaluator is a generic interface for conditions so
// as to provide different implementations based on the operand
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
