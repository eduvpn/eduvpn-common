// Package log implements a basic level based logger
package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"

	"github.com/eduvpn/eduvpn-common/internal/util"
	"github.com/eduvpn/eduvpn-common/types"
)

// FileLogger defines the type of logger that this package implements
// As the name suggests, it saves the log to a file.
type FileLogger struct {
	// Level indicates which maximum level this logger actually forwards to the file
	Level Level

	// file represents a pointer to the open log file
	file  *os.File
}

type Level int8

const (
	// LevelNotSet indicates level not set, not allowed.
	LevelNotSet Level = iota

	// LevelDebug indicates that the message is not an error but is there for debugging.
	LevelDebug

	// LevelInfo indicates that the message is not an error but is there for additional information.
	LevelInfo

	// LevelWarning indicates only a warning, the app still functions.
	LevelWarning

	// LevelError indicates a generic error, the app still functions but some functionality might not work.
	LevelError

	// LevelFatal indicates a fatal error, the app cannot function correctly when such an error occurs.
	LevelFatal
)

// String returns the string of each level.
func (e Level) String() string {
	switch e {
	case LevelNotSet:
		return "NOTSET"
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarning:
		return "WARNING"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Init initializes the logger by forwarding a max level 'level' and a directory 'directory' where the log should be stored
// If the logger cannot be initialized, for example an error in opening the log file, an error is returned.
func (logger *FileLogger) Init(level Level, directory string) error {
	errorMessage := "failed creating log"

	configDirErr := util.EnsureDirectory(directory)
	if configDirErr != nil {
		return types.NewWrappedError(errorMessage, configDirErr)
	}
	logFile, logOpenErr := os.OpenFile(
		logger.filename(directory),
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0o666,
	)
	if logOpenErr != nil {
		return types.NewWrappedError(errorMessage, logOpenErr)
	}
	multi := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multi)
	logger.file = logFile
	logger.Level = level
	return nil
}

// Inherit logs an error with a label using the error level of the error.
func (logger *FileLogger) Inherit(label string, err error) {
	level := types.ErrorLevel(err)

	msg := fmt.Sprintf("%s with err: %s", label, types.ErrorTraceback(err))
	switch level {
	case types.ErrInfo:
		logger.Info(msg)
	case types.ErrWarning:
		logger.Warning(msg)
	case types.ErrOther:
		logger.Error(msg)
	case types.ErrFatal:
		logger.Fatal(msg)
	}
}

// Debug logs a message with parameters as level LevelDebug.
func (logger *FileLogger) Debug(msg string, params ...interface{}) {
	logger.log(LevelDebug, msg, params...)
}

// Debug logs a message with parameters as level LevelInfo.
func (logger *FileLogger) Info(msg string, params ...interface{}) {
	logger.log(LevelInfo, msg, params...)
}

// Debug logs a message with parameters as level LevelWarning.
func (logger *FileLogger) Warning(msg string, params ...interface{}) {
	logger.log(LevelWarning, msg, params...)
}

// Debug logs a message with parameters as level LevelError.
func (logger *FileLogger) Error(msg string, params ...interface{}) {
	logger.log(LevelError, msg, params...)
}

// Debug logs a message with parameters as level LevelFatal.
func (logger *FileLogger) Fatal(msg string, params ...interface{}) {
	logger.log(LevelFatal, msg, params...)
}

// Close closes the logger by closing the internal file.
func (logger *FileLogger) Close() {
	logger.file.Close()
}

// filename returns the filename of the logger by returning the full path as a string.
func (logger *FileLogger) filename(directory string) string {
	return path.Join(directory, "log")
}

// log logs as level 'level' a message 'msg' with parameters 'params'.
func (logger *FileLogger) log(level Level, msg string, params ...interface{}) {
	if level >= logger.Level && logger.Level != LevelNotSet {
		formattedMsg := fmt.Sprintf(msg, params...)
		format := fmt.Sprintf("- Go - %s - %s", level.String(), formattedMsg)
		// To log file
		log.Println(format)
	}
}
