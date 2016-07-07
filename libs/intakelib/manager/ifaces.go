package manager

import (
	"strconv"

	"github.com/gogo/protobuf/proto"
)

// questionAnswerDataSource is an interface used to represent the source
// of authority for current information about questions and answers.
type questionAnswerDataSource interface {
	question(questionID string) question
	answerForQuestion(questionID string) patientAnswer
	valueForKey(key string) interface{}
	registerNode(node layoutUnit, dependencies []layoutUnit)
	deregisterNode(node layoutUnit)
	registerSubscreensForQuestion(q question, subscreens []screen) error
	deregisterSubscreensForQuestion(q question) error
	subscribeForSetupCompletionNotification(listener func() error) error
	clientPlatform() platform
}

// visibility is used to communicate whether a particular
// layoutUnit is hidden or visible. The zero value of
// the visibility type is visible.
type visibility int

const (
	visible visibility = iota
	hidden
)

func (v visibility) String() string {
	switch v {
	case visible:
		return "visible"
	case hidden:
		return "hidden"
	}

	return strconv.Itoa(int(v))
}

// layoutUnit represents any "node" in a layout object
// that is to be sent to the client based on whether or not it is visible.
// Each such node is expected to have a unique and complete descriptor.
type layoutUnit interface {
	layoutParent() layoutUnit
	setLayoutParent(node layoutUnit)

	condition() condition
	setCondition(cond condition)
	children() []layoutUnit
	setLayoutUnitID(id string)

	// descriptor is the string to use to represent the particular layout unit
	// when computing the layoutUnitID.
	descriptor() string

	// layoutUnitID represents a unique ID for a given layoutUnit
	// object.
	layoutUnitID() string

	// unmarshalMapFromClient consumes a very particular client representation
	// of the object from the datamap.
	unmarshalMapFromClient(data dataMap, parent layoutUnit, dataSource questionAnswerDataSource) error

	setVisibility(v visibility)
	visibility() visibility
}

// hasLayoutUnitDependencies is an interface conformed to by objects
// that have layoutUnit dependencies (such as conditions, questions, screens, questions).
type hasLayoutUnitDependencies interface {
	layoutUnitDependencies(dataSource questionAnswerDataSource) []layoutUnit
}

// requirementsValidator is an interface to be conformed to by any
// type to inspect whether or not its requirements have been met.
type requirementsValidator interface {
	requirementsMet(questionAnswerDataSource) (bool, error)
}

// answerValidator is an interface to be conformed to by any type
// to determine whether the patient answer is valid or not.
type answerValidator interface {
	validateAnswer(patientAnswer) error
}

// protobufTransformer is an interface implemented by objects to convert themselves it into a protobuf message.
type protobufTransformer interface {
	transformToProtobuf() (proto.Message, error)
}

// protobufUnmarshaller is an interface implemented by objects to unmarshal the protocol buffer representation
// of themselves.
type protobufUnmarshaller interface {
	unmarshalProtobuf(data []byte) error
}

// mapClientUnmarshaller is the interface implemented by objects that can unmarshal a very particular
// client representation of themselves stored in a map. Note that this is not to be mistaken for the json
// representation of the object as it is possible that the structure of the dataMap differs
// from the pure json representation of the object.
type mapClientUnmarshaller interface {
	unmarshalMapFromClient(data dataMap) error
}

// clientTransformer is the interface implemented by objects that can transform
// themselves into a very specific JSON representation for the client.
type clientTransformer interface {
	transformForClient() (interface{}, error)
}

// condition is an interface to be conformed to by any type
// that can evaluate to true or falsed on its own information
// and the information in the data source.
type condition interface {
	questionIDs() []string
	evaluate(questionAnswerDataSource) bool
	mapClientUnmarshaller
	hasLayoutUnitDependencies
	staticInfoCopier
}

type typed interface {
	TypeName() string
}

// staticInfoCopier is an interface conformed to by objects
// that enable copying of their static info into an a new instance
// of the same type as the object.
type staticInfoCopier interface {
	staticInfoCopy(context map[string]string) interface{}
}

// question is an interface to be conformed to by all
// question types to be used in the visit layout to process incoming
// information from the patient (patient answers) and determine valid answers,
// requirements being met, etc.
type question interface {
	typed
	layoutUnit
	protobufTransformer
	answerValidator
	requirementsValidator
	staticInfoCopier
	stringIndenter

	id() string
	setID(id string)
	prefilled() bool
	patientAnswer() (patientAnswer, error)
	setPatientAnswer(pa patientAnswer) error
	answerForClient() (interface{}, error)

	setParentQuestion(q question)

	// parentQuestion returns the parent question for a subquestion.
	// In all other cases, it returns nil.
	parentQuestion() question

	// canPersisiAnswer returns true if there exists an answer to the question
	// that can be persisted if need be. For instance, it will return false for a photo
	// question if there are photos still being uploaded by the client.
	canPersistAnswer() bool

	String() string
}

// uploadableItem represents any item that will be uploaded by the client
// and then have its serverID communicated to the shared library.
type uploadableItem interface {
	replaceID(idReplacement interface{}) error
	itemUploaded() bool
}

// uploadableItemsContainer represents any object that holds
// uploadableItems
type uploadableItemsContainer interface {
	itemsToBeUploaded() map[string]uploadableItem
}

// screen is an interface to be conformed to be all screen types used in the visit
// layout.
type screen interface {
	typed
	layoutUnit
	protobufTransformer
	requirementsValidator
	staticInfoCopier
	stringIndenter
}

// questionsContainer is an interface conformed by objects that contain questions
type questionsContainer interface {
	questions() []question
}

// patientAnswer is an interface to be conformed to be all patient answer types
// to store answers to questions and pass them across the wire.
type patientAnswer interface {
	mapClientUnmarshaller
	protobufUnmarshaller
	clientTransformer
	protobufTransformer
	stringIndenter
	equals(pa patientAnswer) bool
	isEmpty() bool
}

type titler interface {
	title() string
}

type stringIndenter interface {
	stringIndent(indent string, depth int) string
}
