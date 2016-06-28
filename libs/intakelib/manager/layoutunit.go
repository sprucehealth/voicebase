package manager

import "strconv"

// setLayoutUnitIDForNode recursively walks through the tree structure
// to set the layoutUnitID for the node and all its children.
func setLayoutUnitIDForNode(l layoutUnit, index int, prefix string) {
	layoutUnitID := prefix + l.descriptor() + ":" + strconv.Itoa(index)
	l.setLayoutUnitID(layoutUnitID)

	children := l.children()
	if len(children) > 0 {
		for i, child := range children {
			setLayoutUnitIDForNode(child, i, layoutUnitID+"|")
		}
	}
}

// registerNodeAndDependencies recursively walks through the tree structure
// to register the node's dependencies with the data source, and then recursively
// does the same for its children.
func registerNodeAndDependencies(node layoutUnit, dataSource questionAnswerDataSource) {
	dependencies := computeNodeDependencies(node, dataSource)
	dataSource.registerNode(node, dependencies)

	for _, child := range node.children() {
		registerNodeAndDependencies(child, dataSource)
	}
}

// deregisterNodeAndChildren recursively walks through the tree structure
// of the node to first deregister the children from the datasource and then to deregister
// itself from the datasource.
func deregisterNodeAndChildren(node layoutUnit, dataSource questionAnswerDataSource) {
	for _, child := range node.children() {
		deregisterNodeAndChildren(child, dataSource)
	}

	dataSource.deregisterNode(node)
}

// computeDependencies populates the dependencies for the provided node.
func computeNodeDependencies(node layoutUnit, dataSource questionAnswerDataSource) []layoutUnit {

	var dependencies []layoutUnit
	if cond := node.condition(); cond != nil {
		dependencies = make([]layoutUnit, len(cond.layoutUnitDependencies(dataSource)))
		for i, dependency := range cond.layoutUnitDependencies(dataSource) {
			dependencies[i] = dependency
		}
	}

	if parent := node.layoutParent(); parent != nil {
		dependencies = append(dependencies, parent)
	}

	for _, child := range node.children() {
		dependencies = append(dependencies, child)
	}

	return dependencies
}

// computeLayoutVisibility is used to compute whether or not the layoutUnit is visible based on the
// current information from the datasource, the state of the layoutUnit's parent,
// evaluation of the layoutUnit's condition and its chlidren.
// Note that a current shortcoming of the visibility evaluation is that the children
// are only assessed for their condition versus the overall state of the container (if each
// child in turn contains children). This is something that will need to be improved over time.
func computeLayoutVisibility(lUnit layoutUnit, dataSource questionAnswerDataSource) visibility {

	if lUnit.layoutParent() != nil {
		parentVisibility := computeLayoutVisibility(lUnit.layoutParent(), dataSource)
		if parentVisibility == hidden {
			return hidden
		}
	}

	// if the condition associated with the layoutUnit evaluates to false,
	// it is considered invisible.
	if lUnit.condition() != nil {
		if !lUnit.condition().evaluate(dataSource) {
			return hidden
		}
	}

	// atleast one of the layoutUnit's children have its condition met
	// for the layoutUnit to be visible. If the layoutUnit has no children,
	// then it is considered visible.
	children := lUnit.children()
	atleastOneChildVisible := (children == nil)
	for _, child := range children {
		if child.condition() == nil || child.condition().evaluate(dataSource) {
			atleastOneChildVisible = true
		}
	}

	if atleastOneChildVisible {
		return visible
	}

	return hidden
}
