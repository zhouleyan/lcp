package rest

// sortableCurlyRoutes orders by most parameters and path elements first.
type sortableCurlyRoutes []*Route

func (s sortableCurlyRoutes) Len() int {
	return len(s)
}
func (s sortableCurlyRoutes) Swap(i, j int) {
	(s)[i], (s)[j] = (s)[j], (s)[i]
}
func (s sortableCurlyRoutes) Less(i, j int) bool {
	a := (s)[j]
	b := (s)[i]

	// primary key
	if a.staticCount < b.staticCount {
		return true
	}
	if a.staticCount > b.staticCount {
		return false
	}
	// secondary key
	if a.paramCount < b.paramCount {
		return true
	}
	if a.paramCount > b.paramCount {
		return false
	}
	return a.Path < b.Path
}
