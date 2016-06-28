package manager

// Client is responsible for performing the necessary action for when the
// Manager indicates that an answer to a question is complete and can be committed.
// Examples of actions that the client can take are to upload the answer to the server, persist the answer
// to a database, etc.
type Client interface {

	// PersistAnswerForQuestion is called by the Manager for a question in the following situations:
	// - Valid answer is set for a text-based question
	// - Anwers to subquestions are added for top-level answers to a question to save progress for subquestions.
	// - All subquestions for parent answers to a top-level question have been answered.
	// - All photo slots for a photo question have IDs that are considered complete
	// - A question in review mode is saved.
	PersistAnswerForQuestion(data []byte) error
}
