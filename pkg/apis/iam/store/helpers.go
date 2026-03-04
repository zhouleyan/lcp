package store

// toNullString converts an empty string to nil, otherwise returns a pointer to the string.
func toNullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
