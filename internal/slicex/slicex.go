package slicex

import "golang.org/x/exp/slices"

// Filter returns a new slice containing all elements of s for which f returns true.
func Filter[S ~[]E, E any](s S, fn func(E) bool) S {
	if s == nil {
		return nil
	}

	out := make(S, 0, len(s))
	for _, v := range s {
		if fn(v) {
			out = append(out, v)
		}
	}

	return out
}

// ContainsFunc returns true if s contains an element for which f returns true.
func ContainsFunc[E any](s []E, fn func(E) bool) bool {
	return slices.IndexFunc(s, fn) > -1
}

// Map returns a new slice containing the results of applying f to each element of s.
func Map[E, R any](s []E, fn func(E) R) []R {
	out := make([]R, len(s))
	for i, v := range s {
		out[i] = fn(v)
	}
	return out
}
