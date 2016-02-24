package directory

// EntityIDs is a convenience method for retrieving ID's from a list
// Note: This could be made more gneeric using reflection but don't want the performance cost
func EntityIDs(es []*Entity) []string {
	ids := make([]string, len(es))
	for i, e := range es {
		ids[i] = e.ID
	}
	return ids
}
