package main

// Returns a string with the name of people who still need to answer, excluding people from the second slice
func formatUsers(groupIDs []string, answeredIDs []string) string {
	var s string
	for _, d := range groupIDs {
		if !contains(answeredIDs, d) {
			s += getNickname(d) + ", "
		}
	}

	// Deletes the last comma
	return s[:len(s)-2]
}

// Returns true if the slice contains the element d
func contains(slice []string, d string) bool {
	for _, u := range slice {
		if u == d {
			return true
		}
	}

	return false
}

// Returns the original slice minus the element str
func removeString(slice []string, str string) []string {
	for i, v := range slice {
		if v == str {
			return append(slice[:i], slice[i+1:]...)
		}
	}

	return slice
}
