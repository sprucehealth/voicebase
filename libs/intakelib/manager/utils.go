package manager

import (
	"crypto/rand"
	"fmt"
)

// generateUUID generates a UUID in the standard form.
// code from https://groups.google.com/forum/#!msg/golang-nuts/d0nF_k4dSx4/rPGgfXv6QCoJ
func generateUUID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

func indentAtDepth(indent string, depth int) string {
	b := make([]byte, 0, len(indent)*(depth+1))
	for i := 0; i < (depth + 1); i++ {
		b = append(b, []byte(indent)...)
	}
	return string(b)
}

func parentScreenForQuestion(q question) (*questionScreen, error) {
	if q.layoutParent() == nil {
		return nil, fmt.Errorf("question %s has no parent when one is expected", q.layoutUnitID())
	}

	// identify the screen to which to add the subscreens
	qs, ok := q.layoutParent().(*questionScreen)
	if !ok {
		return nil, fmt.Errorf("expected the question (%s) to be on a question screen but it had %T(%s) as its parent instead.",
			q.layoutUnitID(), q.layoutParent(), q.layoutParent().layoutUnitID())
	}

	return qs, nil
}

func parentSectionForScreen(s screen) (*section, error) {
	if s.layoutParent() == nil {
		return nil, fmt.Errorf("expected screen %s to have a section as its parent but it has no parent.", s.layoutUnitID())
	}

	// possible that the screen is a subscreen, in which case identify the section of the parent screen
	// todo(kajham): note that this assumes that we have subscreens just one layer deep but is a fine assumption for now.
	parentQuestion, ok := s.layoutParent().(question)
	if ok {
		s = parentQuestion.layoutParent().(screen)
	}

	se, ok := s.layoutParent().(*section)
	if !ok {
		return nil, fmt.Errorf("expected screen %s to have a section as its parent but it was %T(%s) instead.",
			s.layoutUnitID(), s.layoutParent(), s.layoutParent().layoutUnitID())
	}

	return se, nil
}
