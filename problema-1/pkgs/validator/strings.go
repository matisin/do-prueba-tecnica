// Package validator provides a set of functions to validate various types of strings.
package validator

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	// DigitRX is a compiled regular expression for matching one or more digits.
	DigitRX = regexp.MustCompile(`^[0-9]+$`)
	// RutRX is a regular expression that matches the format of a Chilean RUT number.
	RutRX = regexp.MustCompile(`^[0-9]{1,8}-[0-9Kk]$`)
	// UUIDRX is a regular expression for matching UUIDs.
	UUIDRX = regexp.MustCompile("^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$")
	// EmailRX is a regular expression for validating email addresses. It uses the standard syntax defined by RFC 5322 and includes support for quoted strings and dotless domains.
	EmailRX = regexp.MustCompile("^[a-zA-Z0-9.!#$%%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
)

// NotEmpty returns true if the trimmed version of the input string is not empty.
func NotEmpty(value string) bool {
	return strings.TrimSpace(value) != ""
}

// MaxChar returns true if the length of the UTF-8 encoded string `value` is less than or equal to `n`.
func MaxChar(value string, n int) bool {
	return utf8.RuneCountInString(value) <= n
}

// MinChar returns true if the length of the UTF-8 encoded string `value` is greater than or equal to `n`.
func MinChar(value string, n int) bool {
	return utf8.RuneCountInString(value) >= n
}

// StringUUID checks if the given string matches the standard UUID format.
func StringUUID(value string) bool {
	match := UUIDRX.MatchString(value)
	return match
}

// StringDigit checks if the given string contains only digits.
func StringDigit(value string) bool {
	match := DigitRX.MatchString(value)
	return match
}

// StringRut checks if the given string matches the Chilean RUT format.
func StringRut(value string) bool {
	match := RutRX.MatchString(value)
	return match
}

// StringEmail checks if the given string is a valid email address.
func StringEmail(value string) bool {
	match := EmailRX.MatchString(value)
	return match
}
