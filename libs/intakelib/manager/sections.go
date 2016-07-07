package manager

type section struct {
	ID           string     `json:"-"`
	LayoutUnitID string     `json:"-"`
	Parent       layoutUnit `json:"-"`
	Title        string     `json:"title"`
	Screens      []screen   `json:"screens"`

	v visibility
}

const (
	sectionTypeScreenContainer = "section_type_screen_container"
)

func (s *section) unmarshalMapFromClient(data dataMap, parent layoutUnit, dataSource questionAnswerDataSource) error {
	if err := data.requiredKeys(sectionTypeScreenContainer, "title", "screens"); err != nil {
		return err
	}

	s.Parent = parent
	s.Title = data.mustGetString("title")

	screens, err := data.getInterfaceSlice("screens")
	if err != nil {
		return err
	}

	s.Screens = make([]screen, len(screens))
	for i, screenVal := range screens {
		screenMap, err := getDataMap(screenVal)
		if err != nil {
			return err
		}

		s.Screens[i], err = getScreen(screenMap, s, dataSource)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *section) descriptor() string {
	return sectionDescriptor
}

func (s *section) title() string {
	return s.Title
}

func (s *section) layoutParent() layoutUnit {
	return nil
}

func (s *section) setLayoutParent(node layoutUnit) {
	s.Parent = node
}

func (s *section) children() []layoutUnit {
	children := make([]layoutUnit, len(s.Screens))
	for i, screen := range s.Screens {
		children[i] = screen
	}

	return children
}

func (s *section) setLayoutUnitID(str string) {
	s.LayoutUnitID = str
}

func (s *section) layoutUnitID() string {
	return s.LayoutUnitID
}

func (s *section) condition() condition {
	return nil
}

func (s *section) setCondition(cond condition) {}

func (s *section) setVisibility(v visibility) {
	s.v = v
}

func (s *section) visibility() visibility {
	return s.v
}

// requirementsMet returns no error if all the contained screens
// have their requirements met.
func (s *section) requirementsMet(dataSource questionAnswerDataSource) (bool, error) {
	for _, sc := range s.Screens {
		if res, err := sc.requirementsMet(dataSource); err != nil {
			return res, err
		} else if !res {
			return false, nil
		}
	}

	return true, nil
}
