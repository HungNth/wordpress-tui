package tui

// UpdateAccumulatedSelection updates the accumulated set based on the user's choices in the current matches view.
// It retains selections from previous queries that are not in the current view, adds newly checked items,
// and deletes items from the current view that the user unchecked.
func UpdateAccumulatedSelection(accumulated map[string]bool, currentMatches []string, chosenInView []string) {
	checked := make(map[string]bool, len(chosenInView))
	for _, s := range chosenInView {
		if s != SearchOptionKey {
			checked[s] = true
			accumulated[s] = true
		}
	}

	for _, m := range currentMatches {
		if !checked[m] && accumulated[m] {
			delete(accumulated, m)
		}
	}
}
