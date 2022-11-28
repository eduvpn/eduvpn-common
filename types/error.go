package types

import (
	"errors"
	"fmt"
)

type ErrLevel int8

const (
	// All other errors, default
	ErrOther ErrLevel = iota

	// The erorr is just here as additional info
	ErrInfo

	// The error is just here as a warning
	ErrWarning

	// The error is fatal, the app cannot function
	ErrFatal
)

type WrappedErrorMessage struct {
	Level   ErrLevel
	Message string
	Err     error
}

// NewWrappedError returns a WrappedErrorMessage and uses the error level from the parent
func NewWrappedError(message string, err error) *WrappedErrorMessage {
	return &WrappedErrorMessage{Level: ErrorLevel(err), Message: message, Err: err}
}

// NewWrappedError returns a WrappedErrorMessage and uses the given error level from the parent
func NewWrappedErrorLevel(level ErrLevel, message string, err error) *WrappedErrorMessage {
	return &WrappedErrorMessage{Level: level, Message: message, Err: err}
}

func (e *WrappedErrorMessage) Unwrap() error {
	return e.Err
}

func (e *WrappedErrorMessage) Cause() error {
	causeErr := e.Err
	for errors.Unwrap(causeErr) != nil {
		causeErr = errors.Unwrap(causeErr)
	}
	return causeErr
}

func (e *WrappedErrorMessage) Traceback() string {
	returnStr := fmt.Sprintf("%s\n%s", e.Message, "Traceback:")
	causeErr := e.Err
	for errors.Unwrap(causeErr) != nil {
		causeErr = errors.Unwrap(causeErr)
		var wrappedErr *WrappedErrorMessage

		errorStr := causeErr.Error()

		if errors.As(causeErr, &wrappedErr) {
			errorStr = wrappedErr.Message
		}
		returnStr += fmt.Sprintf("\n - %s", errorStr)
	}
	return returnStr
}

func (e *WrappedErrorMessage) Error() string {
	return fmt.Sprintf("Got error: %s, with cause: %s", e.Message, e.Err)
}

func ErrorTraceback(err error) string {
	var wrappedErr *WrappedErrorMessage

	if errors.As(err, &wrappedErr) {
		return wrappedErr.Traceback()
	}
	return err.Error()
}

func ErrorCause(err error) error {
	var wrappedErr *WrappedErrorMessage

	if errors.As(err, &wrappedErr) {
		return wrappedErr.Cause()
	}
	return err
}

func ErrorLevel(err error) ErrLevel {
	var wrappedErr *WrappedErrorMessage

	if errors.As(err, &wrappedErr) {
		return wrappedErr.Level
	}
	return ErrOther
}
