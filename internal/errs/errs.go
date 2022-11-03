// Package errs simplifies creation of errors and contains errors produced.
package errs

import "fmt"

// Error type is used to create constant errors.
type Error string

func (e Error) Error() string { return string(e) }

// Errorf creates error from formatted string with params.
func Errorf(format string, v ...interface{}) Error {
	return Error(fmt.Sprintf(format, v...))
}
