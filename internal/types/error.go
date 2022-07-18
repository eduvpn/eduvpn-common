package types

import (
	"encoding/json"
	"errors"
	"fmt"
)

type ErrorLevel int8

const (
	// All other errors
	ERR_OTHER ErrorLevel = iota

	// The error is just here as additional info
	ERR_INFO
)

type WrappedErrorMessage struct {
	Level   ErrorLevel
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

func GetErrorLevel(err error) ErrorLevel {
	var wrappedErr *WrappedErrorMessage

	if errors.As(err, &wrappedErr) {
		return wrappedErr.Level
	}
	return ERR_OTHER
}

type WrappedErrorMessageJSON struct {
	Level     ErrorLevel `json:"level"`
	Cause     string     `json:"cause"`
	Traceback string     `json:"traceback"`
}

func GetErrorJSONString(err error) string {
	var wrappedErr *WrappedErrorMessage

	var level ErrorLevel
	var cause error
	var traceback string

	if errors.As(err, &wrappedErr) {
		level = wrappedErr.Level
		cause = wrappedErr.Cause()
		traceback = wrappedErr.Traceback()
	} else {
		level = ERR_OTHER
		cause = err
		traceback = err.Error()
	}

	json, jsonErr := json.Marshal(&WrappedErrorMessageJSON{Level: level, Cause: cause.Error(), Traceback: traceback})

	if jsonErr != nil {
		panic(jsonErr)
	}
	return string(json)
}
