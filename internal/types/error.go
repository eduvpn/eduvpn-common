package types

import (
	"errors"
	"fmt"
)

type WrappedErrorMessage struct {
	Message string
	Err     error
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
	returnStr := fmt.Sprintf("Traceback for error: %s", e.Message)
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

func GetErrorTraceback(err error) string {
	var wrappedErr *WrappedErrorMessage

	if errors.As(err, &wrappedErr) {
		return wrappedErr.Traceback()
	}
	return err.Error()
}

func GetErrorCause(err error) error {
	var wrappedErr *WrappedErrorMessage

	if errors.As(err, &wrappedErr) {
		return wrappedErr.Cause()
	}
	return err
}
