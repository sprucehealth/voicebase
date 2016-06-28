package manager

// Manager is responsible for the management of answers to questions in a layout.
//
// Specifically, it is responsible for:
// - Accepting answers to questions in a layout.
// - Validating answers to questions.
// - Tracking when answers to a question are considered complete, and informing the client of this.
// - Computing and providing the next screen to go to based on answers to questions.
// - Validating the layout against the answers to questions.
// - Enabling editing of answers to questions while maintaining validity of the layout.
type VisitManager interface {

	// Init initializes the manager with a particular layout and client implementation.
	// If the manager does not understand the layout, it returns an error to indicate so.
	Init(data []byte, cli Client) error

	// Set data enables the client to inform the intakelib of specific type of data
	// that is used in the business logic of the intakelib.
	// Note that while the key is of type string, only particular types of keys are accepted
	// for each layout type supported to ensure that client does not use this as a dumping ground
	// of values. The accepted key types are agreed upon between client and intakelib without any strict
	// enforcing at the interface layer only to ensure that definition is generic for all layout types.
	Set(data []byte) error

	// ComputeNextScreen returns the next logical screenID in the flow based on the currentScreenID
	// and the answers to questions until the point when the method is called. An empty string returns the screenID for the visit overview.
	//
	//
	// Note that the next screenID is only returned if the current screen is considered valid (by internally calling ValidateScreen, else an error is returned.)
	//
	// Also note that specifically for the visit layout,
	// ComputeNextScreen returns screenIDs all the way through the last state of the visit overview, at which point
	// calling ComputeNextScreen would return an empty string. It is the client's responsibiility to do the necessary work
	// at the end of certain screens, such as submit the visit when all requirements are satisified after the checkout screen.
	// Similarly, calling ComputeNextScreen from the triage screen would result in a dead end too.
	ComputeNextScreen(currentScreenID string) ([]byte, error)

	// ValidateScreen returns an object that communicates whether or not the screen is considered valid.
	// A screen is considered valid if all its requirements for the screen have been met
	// (ie, all visible questions on the screen have been answered, or pharmacy has been entered, etc.)
	ValidateScreen(screenID string) ([]byte, error)

	// Screen returns the screen for the given screenID and
	// nil if the screen doesn't exist.
	//
	// This method will be useful for refreshing state of current screen.
	Screen(screenID string) ([]byte, error)

	// SetAnswerForQuestion accepts a valid answer to a question. If the answer is not valid, an error is returned.
	//
	// This is the only way to set answers for questions, regardless of whether the answer is considered
	// partial (such as subquestions or photo slots with local ids), uncommitted (such as review mode) or complete.
	//
	// - Set the complete answer for a particular question and the manager will in turn call PersistAnswerForQuestion
	// - Set the answer for a subquestion for a particular parent answer, and the manager will in turn call PersistAnswerForQuestion
	// to save progress for answers to subquestions.
	// - Set an answer for a particular phtoto section question with local photo ids and then call SetServerIDForPhoto each time the serverID for a photo part of the photo section
	// answer is received. The manager will only call PersistAnswerForQuestion if all photo slots have serverIDs.
	SetAnswerForQuestion(questionID string, data []byte) error

	// ReplaceID uses the idReplacement object to set the newID, along with any other updated information,
	// for the currentID of a particular resource being tracked by the manager.
	// The assumption is that the manager knows of the currentID but if it doesn't it throws an exception.
	ReplaceID(currentID string, idReplacement []byte) error

	// ComputeLayoutStatus computes and returns a layout specific status representation.
	// For the visit layout, for instance, it returns an object that represents the sections and their current status.
	ComputeLayoutStatus() ([]byte, error)

	// ValidateRequirementsInLayout validates the layout against the answers to question
	// and returns a object that representst the result of the validation.
	//
	// This method is called to ensure that all questions have been answered before submitting the layout.
	ValidateRequirementsInLayout() ([]byte, error)

	// StartReviewModeWithQuestion creates the internal bookkeeing necessary to track edit mode
	// for a particular question as a starting point. What is returned is a screen object with an ID
	// that indicates the review mode ID for the screen.
	//
	// Client should then call ComputeNextScreen to get the next screen in edit mode.
	StartEditModeWithQuestion(questionID string) ([]byte, error)

	// EndEditMode either commits or discards the answers to questions in this mode, depending on the discard flag.
	// ValidateRequirementsInLayout is called internally when discard is false to ensure that the answers being committed
	// result in a layout with its requirements satisified.
	EndEditMode(discard int) error
}
