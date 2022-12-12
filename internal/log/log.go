// Package log implements a basic level based logger
package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"

	"github.com/eduvpn/eduvpn-common/internal/oauth"

	"github.com/eduvpn/eduvpn-common/internal/util"
	"github.com/go-errors/errors"
)

type ErrLevel int8

const (
	ErrOther ErrLevel = iota
	ErrInfo
	ErrWarning
	ErrFatal
)

func GetErrorLevel(err error) ErrLevel {
	// Get the inner error
	e := err
	if err1, ok := err.(*errors.Error); ok {
		e = err1.Err
	}

	switch e.(type) {
	case *oauth.CancelledCallbackError:
		return ErrInfo
	default:
		return ErrOther
	}
}

// FileLogger defines the type of logger that this package implements
// As the name suggests, it saves the log to a file.
type FileLogger struct {
	// Level indicates which maximum level this logger actually forwards to the file
	Level Level

	// file represents a pointer to the open log file
	file *os.File
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
func (logger *FileLogger) Init(lvl Level, dir string) error {
	err := util.EnsureDirectory(dir)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(
		logger.filename(dir),
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0o666,
	)
	if err != nil {
		return errors.WrapPrefix(err, "failed creating log", 0)
	}
	multi := io.MultiWriter(os.Stdout, f)
	log.SetOutput(multi)
	logger.file = f
	logger.Level = lvl
	return nil
}

// Inherit logs an error with a label using the error level of the error.
func (logger *FileLogger) Inherit(err error) {
	if err == nil {
		return
	}
	switch GetErrorLevel(err) {
	case ErrInfo:
		logger.Infof(err.Error())
	case ErrWarning:
		logger.Warningf(err.Error())
	case ErrOther:
		logger.Errorf(err.Error())
	case ErrFatal:
		logger.Fatalf(err.Error())
	}
}

// Debugf logs a message with parameters as level LevelDebug.
func (logger *FileLogger) Debugf(msg string, params ...interface{}) {
	logger.log(LevelDebug, msg, params...)
}

// Infof logs a message with parameters as level LevelInfo.
func (logger *FileLogger) Infof(msg string, params ...interface{}) {
	logger.log(LevelInfo, msg, params...)
}

// Warningf logs a message with parameters as level LevelWarning.
func (logger *FileLogger) Warningf(msg string, params ...interface{}) {
	logger.log(LevelWarning, msg, params...)
}

// Errorf logs a message with parameters as level LevelError.
func (logger *FileLogger) Errorf(msg string, params ...interface{}) {
	logger.log(LevelError, msg, params...)
}

// Fatalf logs a message with parameters as level LevelFatal.
func (logger *FileLogger) Fatalf(msg string, params ...interface{}) {
	logger.log(LevelFatal, msg, params...)
}

// Close closes the logger by closing the internal file.
func (logger *FileLogger) Close() error {
	return logger.file.Close()
}

// filename returns the filename of the logger by returning the full path as a string.
func (logger *FileLogger) filename(directory string) string {
	return path.Join(directory, "log")
}

// log logs as level 'level' a message 'msg' with parameters 'params'.
func (logger *FileLogger) log(lvl Level, msg string, params ...interface{}) {
	if lvl >= logger.Level && logger.Level != LevelNotSet {
		fMsg := fmt.Sprintf(msg, params...)
		f := fmt.Sprintf("- Go - %s - %s", lvl.String(), fMsg)
		// To log file
		log.Println(f)
	}
}
