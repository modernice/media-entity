package mapx

// Ensure returns a new initialized map of the provided map is nil.
func Ensure[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		m = make(map[K]V)
	}
	return m
}
