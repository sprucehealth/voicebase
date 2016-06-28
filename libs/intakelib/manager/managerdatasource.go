package manager

import (
	"container/list"
	"fmt"
)

// This file contains all the questionAnswerDatasource interface implementations so that the visitManager
// conforms to the questionAnswerDataSource interface.

func (v *visitManager) question(questionID string) question {

	if qData, ok := v.questionMap[questionID]; ok {
		return qData.questionRef
	}

	return nil
}

func (v *visitManager) valueForKey(key string) []byte {
	return v.userFields.get(key)
}

func (v *visitManager) subscribeForSetupCompletionNotification(listener func() error) error {
	// call the listener immediately if setup is already complete
	if v.setupComplete {
		return listener()
	}
	v.setupCompleteListeners = append(v.setupCompleteListeners, listener)
	return nil
}

func (v *visitManager) notifyListenersOfSetupComplete() error {
	for _, listener := range v.setupCompleteListeners {
		if err := listener(); err != nil {
			return err
		}
	}

	// drain the listeners now that they have been notified
	v.setupCompleteListeners = nil
	return nil
}

func (v *visitManager) registerSubscreensForQuestion(q question, subscreens []screen) error {

	qs, err := parentScreenForQuestion(q)
	if err != nil {
		return err
	}
	qs.registerSubscreensForQuestion(q, subscreens)

	// identify the section within which the screen belongs so as to identify the section-level
	// list of screens into which to insert the subscreens
	se, err := parentSectionForScreen(qs)
	if err != nil {
		return err
	}

	screenList, ok := v.sectionScreensMap[se.layoutUnitID()]
	if !ok {
		return fmt.Errorf("section %s doesn't exist in the map", se.layoutUnitID())
	}

	// identify the position in the screenList after which to insert the subscreens.
	// the starting point will either be the position after the question screen
	// OR, if other questions on the screen have subcreens, then after the last subscreen
	// of the question immediately before the provided question
	insertAfter := screen(qs)
	for _, questionOnScreen := range qs.Questions {
		if questionOnScreen.layoutUnitID() == q.layoutUnitID() {
			break
		}

		subscreensForQuestion := qs.subscreensMap[questionOnScreen.layoutUnitID()]
		if len(subscreensForQuestion) > 0 {
			insertAfter = subscreensForQuestion[len(subscreensForQuestion)-1]
		}
	}

	// identify the element in the list that contains the insertAfter screen
	var insertAfterElement *list.Element
	for e := screenList.Front(); e != nil; e = e.Next() {
		if e.Value.(screen).layoutUnitID() == insertAfter.layoutUnitID() {
			insertAfterElement = e
			break
		}
	}

	if insertAfterElement == nil {
		return fmt.Errorf("Unable to find element to insert subscreens for question %s after in list", q.layoutUnitID())
	}

	// go through the slice of subscreens and insert into the list
	for _, subscreen := range subscreens {
		insertAfterElement = screenList.InsertAfter(subscreen, insertAfterElement)

		// register each subscreen
		registerNodeAndDependencies(subscreen, v)
	}

	return nil
}

func (v *visitManager) deregisterSubscreensForQuestion(q question) error {
	qs, err := parentScreenForQuestion(q)
	if err != nil {
		return err
	}

	subscreensToRemove := qs.subscreensMap[q.layoutUnitID()]
	if len(subscreensToRemove) == 0 {
		return nil
	}
	qs.deregisterSubscreensForQuestion(q)

	// remove the subscreens from the section-level screens
	se, err := parentSectionForScreen(qs)
	if err != nil {
		return err
	}

	screenList, ok := v.sectionScreensMap[se.layoutUnitID()]
	if !ok {
		return fmt.Errorf("section %s does not exist", se.layoutUnitID())
	}

	var elementToRemove *list.Element
	for e := screenList.Front(); e != nil; e = e.Next() {
		if e.Value.(screen).layoutUnitID() == subscreensToRemove[0].layoutUnitID() {
			elementToRemove = e
			break
		}
	}

	if elementToRemove == nil {
		return fmt.Errorf("subscreen %s not found in the screen list.", subscreensToRemove[0].layoutUnitID())
	}

	for _, subscreen := range subscreensToRemove {
		if elementToRemove == nil {
			break
		}

		nextElementToRemove := elementToRemove.Next()
		if subscreen.layoutUnitID() == elementToRemove.Value.(screen).layoutUnitID() {
			screenList.Remove(elementToRemove)
		} else {
			return fmt.Errorf("Expected %s but got %s while removing subscreens.", subscreen.layoutUnitID(), elementToRemove.Value.(screen).layoutUnitID())
		}
		elementToRemove = nextElementToRemove

		// deregister each subscreen as it is removed
		deregisterNodeAndChildren(subscreen, v)
	}

	return nil
}

func (v *visitManager) clientPlatform() platform {
	return v.platform
}
