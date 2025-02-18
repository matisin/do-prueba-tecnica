// Package validator has a set of functions that validates diferent values in various contexts
package validator

import "slices"

type ValidationFunc func(params ...any) bool

// AllowedValues reports whether a value is present in the specified allowed values.
func AllowedValues[T comparable](value T, allowedValues ...T) bool {
	return slices.Contains(allowedValues, value)
}
