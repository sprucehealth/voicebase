package manager

import "testing"

type mockCondition_visible struct {
	condition
	res bool
}

func (m *mockCondition_visible) evaluate(questionAnswerDataSource) bool {
	return m.res
}

type mockLayoutUnit_visible struct {
	layoutUnit
	parent *mockLayoutUnit_visible
	ch     []layoutUnit
	cond   condition
}

func (m *mockLayoutUnit_visible) layoutParent() layoutUnit {
	if m.parent == nil {
		return nil
	}
	return m.parent
}

func (m *mockLayoutUnit_visible) condition() condition {
	return m.cond
}

func (m *mockLayoutUnit_visible) children() []layoutUnit {
	return m.ch
}

func (m *mockLayoutUnit_visible) descriptor() string {
	return ""
}

func (m *mockLayoutUnit_visible) layoutUnitID() string {
	return ""
}

type mockDataSource_visible struct {
	layoutVisibility bool
	questionAnswerDataSource
}

func (m *mockDataSource_visible) question(questionID string) question {
	return nil
}

func TestLayoutVisibility_FalseCondition(t *testing.T) {
	l := &mockLayoutUnit_visible{
		cond: &mockCondition_visible{
			res: false,
		},
		parent: &mockLayoutUnit_visible{
			parent: nil,
		},
	}

	m := &mockDataSource_visible{
		layoutVisibility: true,
	}

	v := computeLayoutVisibility(l, m)
	if v != hidden {
		t.Fatal("Expected layout unit to be hidden but was visible")
	}
}

func TestLayoutVisibility_NoChildren(t *testing.T) {
	l := &mockLayoutUnit_visible{
		cond: &mockCondition_visible{
			res: true,
		},
		parent: &mockLayoutUnit_visible{},
	}

	m := &mockDataSource_visible{
		layoutVisibility: true,
	}

	v := computeLayoutVisibility(l, m)
	if v != visible {
		t.Fatal("Expected layout unit to be visible but was hidden")
	}
}

func TestLayoutVisibility_HiddenChildren(t *testing.T) {
	l := &mockLayoutUnit_visible{
		cond: &mockCondition_visible{
			res: true,
		},
		parent: &mockLayoutUnit_visible{},
		ch: []layoutUnit{
			&mockLayoutUnit_visible{
				cond: &mockCondition_visible{
					res: false,
				},
			},
		},
	}

	m := &mockDataSource_visible{
		layoutVisibility: true,
	}

	v := computeLayoutVisibility(l, m)
	if v != hidden {
		t.Fatal("Expected layout unit to be hidden but was visible")
	}
}

func TestLayoutVisibility_AtleastOneVisibleChild(t *testing.T) {
	l := &mockLayoutUnit_visible{
		cond: &mockCondition_visible{
			res: true,
		},
		parent: &mockLayoutUnit_visible{},
		ch: []layoutUnit{
			&mockLayoutUnit_visible{
				cond: &mockCondition_visible{
					res: false,
				},
			},
			&mockLayoutUnit_visible{
				cond: &mockCondition_visible{
					res: true,
				},
			},
		},
	}

	m := &mockDataSource_visible{
		layoutVisibility: true,
	}

	v := computeLayoutVisibility(l, m)
	if v != visible {
		t.Fatal("Expected layout unit to be visible but was hidden")
	}
}
