package manager

// This file contains all the intake.Manager interface implementations
// by an object of visitManager type.

import (
	"container/list"
	"errors"
	"fmt"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

// clientClock is used to communicate the local point in
// time at which a question was answered by a patient so as to ensure
// that the latest answer is stored server-side.
type clientClock struct {

	// sessionID is a UUID generated at the initialization of the
	// visit manager and represents the session within which the
	// client is interacting with the visit.
	sessionID string

	// sessionCounter is a monotonically increasing
	// counter used to represent a point in time at
	// which a question was answered by the patient.
	sessionCounter uint
}

// questionData holds a reference to the question along
// with any additional book-keeping for question answer management.
type questionData struct {

	// isAnswerDirty indicates whether the answer held by the
	// question is ready to be persisted by the client.
	isAnswerDirty bool

	// prefilledAnswerPersisted indicates whether a question
	// with prefilled answer has had its answer persisted to the client.
	// This tracking is required so that we can
	// treat a prefilled answer as dirty (even if equal) the first time around
	// and then do equality check therafter such that we are not persisting
	// the same prefilled answer over and over again.
	prefilledAnswerPersisted bool

	// questionRef is a reference to the question.
	questionRef question
}

type visitManager struct {
	cli           Client
	platform      platform
	visit         *visit
	serializer    serializer
	userFields    *userFields
	setupComplete bool

	rwMutex sync.RWMutex

	// questionMap is a mapping of the question id  to questionData
	questionMap map[string]*questionData

	// layoutUnitMap is a mapping of layoutUnitID to a layoutUnitNode
	// in the vist object.
	layoutUnitMap map[string]layoutUnit

	// itemsToBeUploaded is a container for all items that have a client local ID
	// and are in the process of being uploaded by the client.
	itemsToBeUploaded map[string]uploadableItem

	// dependantsMap is a container of a layoutUnitID -> list of dependants
	// so as to recursively be able to evaluate all dependants
	// for a given node when an answer to a question is updated.
	dependantsMap map[string][]layoutUnit

	// visitStatus is responsible for keeping an up to date view of the visit
	// status upon an answer or keyValuePair being set by the client.
	visitStatus *visitCompletionStatus

	// listeners is a list of components to notify when the visitManager
	// has finished being setup.
	setupCompleteListeners []func() error

	// clienClock is the manager's state of the client clock.
	clientClock

	// sectionScreensMap is a mapping of section's layoutUnitID to the linear list
	// of real-time potential screens to display. "Real-time" because included in the
	// list is the subscreens based on answers to questions that are to be shown.
	sectionScreensMap map[string]*list.List
}

func (v *visitManager) Init(data []byte, cli Client) error {
	var err error
	var vd visitData

	if err := vd.unmarshal(protobuf, data); err != nil {
		return err
	}
	v.cli = cli

	v.serializer, err = serializerForType(protobuf)
	if err != nil {
		return err
	}

	v.platform = vd.platform

	v.visit = &visit{
		ID:          vd.patientVisitID,
		IsSubmitted: vd.isSubmitted,
	}
	if err := v.visit.unmarshalMapFromClient(vd.layoutData, nil, v); err != nil {
		return err
	}

	v.userFields = vd.userFields

	v.sessionID, err = generateUUID()
	if err != nil {
		return err
	}

	// initialize the questionMap
	v.questionMap = make(map[string]*questionData)
	questions := v.visit.questions()
	for _, q := range questions {
		v.questionMap[q.id()] = &questionData{
			questionRef: q,
		}
	}

	// set unique layout unit ids for each node in the tree
	// so as to make them easy to reference
	v.visit.setLayoutUnitIDsForSections()

	// once the questionMap has been populated for the dataSource
	// and the layoutUnitIDs now exist, register the dependencies
	// of each node with the data source for dependency management
	registerNodeAndDependencies(v.visit, v)

	v.setupStaticScreenForAllSections()

	v.itemsToBeUploaded = make(map[string]uploadableItem)

	if err := v.notifyListenersOfSetupComplete(); err != nil {
		return err
	}

	// now that listeners have been processed, compute overall visit status.
	v.computeCurrentVisibilityState(v.visit)
	v.visitStatus, err = newVisitCompletionStatus(v)
	if err != nil {
		return err
	}
	if err := v.visitStatus.update(); err != nil {
		return err
	}

	v.setupComplete = true
	return nil
}

func (v *visitManager) Set(data []byte) error {
	v.rwMutex.Lock()
	defer v.rwMutex.Unlock()

	var pair intake.KeyValuePair
	if err := proto.Unmarshal(data, &pair); err != nil {
		return err
	}

	if err := v.userFields.set(*pair.Key, pair.Value); err != nil {
		return err
	}

	// update the overall visit status based on the answer set
	return v.visitStatus.update()
}

func (v *visitManager) ComputeNextScreen(currentScreenID string) ([]byte, error) {
	v.rwMutex.RLock()
	defer v.rwMutex.RUnlock()

	var err error

	if currentScreenID == "" {
		return wrapScreenID(screenTypeVisitOverview.String(), screenTypeVisitOverview.String(), v.serializer)
	}

	// only move to the next screen if the current screen has its requirements met
	s, err := v.screen(currentScreenID)
	if err != nil {
		return nil, err
	}

	res, err := s.requirementsMet(v)
	switch {
	case err == errSubQuestionRequirements:
		// this is okay since the only way subquestions will have their requirements
		// met is if we allow that to be the case by moving to the next screen :)
	case err != nil:
		return nil, err
	case !res:
		return nil, errors.New("Cannot move to next screen until current screen has its requirements met.")
	}

	se, err := parentSectionForScreen(s)
	if err != nil {
		return nil, err
	}

	nextScreen, err := v.computeNextScreenInSection(se, s)
	if err != nil {
		return nil, err
	} else if nextScreen == nil {
		return wrapScreenID(screenTypeVisitOverview.String(), screenTypeVisitOverview.String(), v.serializer)
	}

	return wrapScreenID(nextScreen.layoutUnitID(), nextScreen.TypeName(), v.serializer)
}

func (v *visitManager) ValidateScreen(screenID string) ([]byte, error) {
	v.rwMutex.RLock()
	defer v.rwMutex.RUnlock()

	s, err := v.screen(screenID)
	if err != nil {
		return nil, err
	}

	return validateScreen(s, v, v.serializer)
}

func (v *visitManager) Screen(screenID string) ([]byte, error) {
	v.rwMutex.RLock()
	defer v.rwMutex.RUnlock()

	if screenID == screenTypeVisitOverview.String() {
		return createMarshalledVisitOverviewScreen(v.visitStatus)
	}

	s, err := v.screen(screenID)
	if err != nil {
		return nil, err
	}

	if s.visibility() == hidden {
		return nil, fmt.Errorf("Cannot request for a hidden screen (id = %s)", screenID)
	}
	progress, err := v.computeProgress(s)
	if err != nil {
		return nil, err
	}

	return wrapScreen(s, progress, v.serializer)
}

func (v *visitManager) SetAnswerForQuestion(questionID string, data []byte) error {
	v.rwMutex.Lock()
	defer v.rwMutex.Unlock()

	if v.visit.IsSubmitted {
		return errVisitReadOnlyMode
	}

	var ad answerData
	if err := ad.unmarshalProtobuf(data); err != nil {
		return err
	}

	// check if there already exists an answer for the question
	qd := v.questionMap[questionID]
	if qd == nil {
		return fmt.Errorf("question %s doesn't exist in layout", questionID)
	}

	existingPatientAnswer, err := qd.questionRef.patientAnswer()
	if err != errNoAnswerExists && err != nil {
		return err
	}

	// nothing to do if the answers are equal
	if err == nil &&
		(!qd.questionRef.prefilled() || qd.prefilledAnswerPersisted) &&
		existingPatientAnswer.equals(ad.answer) {
		return nil
	}

	// set the questionID for the answer so that the answer is able
	// to marshal itself into a complete object for the client to persist
	ad.answer.setQuestionID(questionID)

	if err := qd.questionRef.setPatientAnswer(ad.answer); err != nil {
		return err
	}

	// mark as dirty if answer was successfully set
	qd.isAnswerDirty = true

	// track any items that are to be uploaded by the client
	container, ok := ad.answer.(uploadableItemsContainer)
	if ok {
		items := container.itemsToBeUploaded()
		for localID, item := range items {
			v.itemsToBeUploaded[localID] = item
		}
	}

	v.evaluateDependants(qd.questionRef)

	v.persistAllDirtyQuestions()

	// update the overall visit status based on the answer set
	if err := v.visitStatus.update(); err != nil {
		return err
	}

	return nil
}

func (v *visitManager) ReplaceID(currentID string, data []byte) error {
	v.rwMutex.Lock()
	defer v.rwMutex.Unlock()

	if v.visit.IsSubmitted {
		return errVisitReadOnlyMode
	}

	item, ok := v.itemsToBeUploaded[currentID]
	if !ok {
		return fmt.Errorf("Item with currentID %s does not exist", currentID)
	}

	var id idReplacementData
	if err := id.unmarshalProtobuf(data); err != nil {
		return err
	}

	if err := item.replaceID(id.replacementData); err != nil {
		return err
	}

	delete(v.itemsToBeUploaded, currentID)
	v.persistAllDirtyQuestions()

	return nil
}

func (v *visitManager) ComputeLayoutStatus() ([]byte, error) {
	v.rwMutex.Lock()
	defer v.rwMutex.Unlock()

	pb, err := v.visitStatus.transformToProtobuf()
	if err != nil {
		return nil, err
	}

	return v.serializer.marshal(pb)
}

func (v *visitManager) ValidateRequirementsInLayout() ([]byte, error) {
	v.rwMutex.RLock()
	defer v.rwMutex.RUnlock()

	return v.visit.validateRequirements(v, v.serializer)
}

func (v *visitManager) StartEditModeWithQuestion(questionID string) ([]byte, error) {
	v.rwMutex.Lock()
	defer v.rwMutex.Unlock()

	if v.visit.IsSubmitted {
		return nil, errVisitReadOnlyMode
	}
	return nil, nil
}

func (v *visitManager) EndEditMode(discard int) error {
	v.rwMutex.Lock()
	defer v.rwMutex.Unlock()

	if v.visit.IsSubmitted {
		return errVisitReadOnlyMode
	}

	return nil
}
