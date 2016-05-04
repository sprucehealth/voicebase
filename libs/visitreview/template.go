package visitreview

import "fmt"

const (
	ConditionKeyExists    = "key_exists"
	ConditionAnyKeyExists = "any_key_exists"
)

// ViewContext is a generic container for data that Views consume
// from when being rendered. Data exists in the ViewContext based on keys
// which are specified in the "content_config" of a view definition in the template
type ViewContext struct {
	context           map[string]interface{}
	IgnoreMissingKeys bool
}

func NewViewContext(context map[string]interface{}) *ViewContext {
	if context == nil {
		context = make(map[string]interface{})
	}

	return &ViewContext{
		context: context,
	}
}

func (c *ViewContext) Set(key string, data interface{}) {
	c.context[key] = data
}

func (c *ViewContext) Get(key string) (interface{}, bool) {
	data, ok := c.context[key]
	return data, ok
}

func (c *ViewContext) Delete(key string) {
	delete(c.context, key)
}

// Typed is an interface implemented by structs that can return their own type name
type Typed interface {
	TypeName() string
}

type View interface {
	Typed
	Render(context *ViewContext) (map[string]interface{}, error)
	Validate() error
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

// ViewCondition is a structure found in the ContentConfig
// of views. The Operand defines the type of implementation to use
// to evaluate the condition based on the key, to either true or false
// which essentially indicates whether or not to include the view in the rendering
type ViewCondition struct {
	Op   string   `json:"op,omitempty"`
	Key  string   `json:"key,omitempty"`
	Keys []string `json:"keys,omitempty"`
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
	EvaluateCondition(condition ViewCondition, context *ViewContext) (bool, error)
}

type KeyExistsEvaluator int64

func (d KeyExistsEvaluator) EvaluateCondition(condition ViewCondition, context *ViewContext) (bool, error) {
	if condition.Op != ConditionKeyExists {
		return false, ViewConditionEvaluationError{Message: fmt.Sprintf("Condition evaluation called with wrong operand. Expected key_exists but got %s", condition.Op)}
	}

	_, ok := context.Get(condition.Key)
	return ok, nil
}

type AnyKeyExistsEvaluator int64

func (d AnyKeyExistsEvaluator) EvaluateCondition(condition ViewCondition, context *ViewContext) (bool, error) {
	if condition.Op != ConditionAnyKeyExists {
		return false, ViewConditionEvaluationError{Message: fmt.Sprintf("Expected operand any_key_exists but got %s", condition.Op)}
	}

	for _, key := range condition.Keys {
		_, ok := context.Get(key)
		if ok {
			return true, nil
		}
	}

	return false, nil
}

var conditionEvaluators = make(map[string]ConditionEvaluator)

func init() {
	conditionEvaluators[ConditionKeyExists] = KeyExistsEvaluator(0)
	conditionEvaluators[ConditionAnyKeyExists] = AnyKeyExistsEvaluator(0)
}

func EvaluateConditionForView(view View, condition ViewCondition, context *ViewContext) (bool, error) {
	conditionEvaluator, ok := conditionEvaluators[condition.Op]
	if !ok {
		return false, ViewConditionEvaluationError{Message: fmt.Sprintf("Unable to find condition with op %s for view type %s", condition.Op, view.TypeName())}
	}

	return conditionEvaluator.EvaluateCondition(condition, context)
}
