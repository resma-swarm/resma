package confirm

// Confirm prompts the user for confirmation.
// If skip is true, it returns true without prompting.
func Confirm(prompt string, skip bool) (bool, error) {
	if skip {
		return true, nil
	}
	// Stub implementation
	return false, nil
}
