package log

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/eduvpn/eduvpn-common/internal/util"
	"github.com/eduvpn/eduvpn-common/types"
)

type FileLogger struct {
	Level LogLevel
	File  *os.File
}

type LogLevel int8

const (
	// No level set, not allowed
	LOG_NOTSET LogLevel = iota
	// Log info, this message is not an error
	LOG_INFO
	// Log only to provide a warning, the app still functions
	LOG_WARNING
	// Log to provide a generic error, the app still functions but some functionality might not work
	LOG_ERROR
	// Log to provide a fatal error, the app cannot function correctly when such an error occurs
	LOG_FATAL
)

func (e LogLevel) String() string {
	switch e {
	case LOG_NOTSET:
		return "NOTSET"
	case LOG_INFO:
		return "INFO"
	case LOG_WARNING:
		return "WARNING"
	case LOG_ERROR:
		return "ERROR"
	case LOG_FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

func (logger *FileLogger) Init(level LogLevel, name string, directory string) error {
	errorMessage := "failed creating log"

	configDirErr := util.EnsureDirectory(directory)
	if configDirErr != nil {
		return types.NewWrappedError(errorMessage, configDirErr)
	}
	logFile, logOpenErr := os.OpenFile(
		logger.getFilename(directory, name),
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0o666,
	)
	if logOpenErr != nil {
		return types.NewWrappedError(errorMessage, logOpenErr)
	}
	log.SetOutput(logFile)
	logger.File = logFile
	logger.Level = level
	return nil
}

func (logger *FileLogger) Inherit(label string, err error) {
	level := types.GetErrorLevel(err)

	msg := fmt.Sprintf("%s with err: %s", label, types.GetErrorTraceback(err))
	switch level {
	case types.ERR_INFO:
		logger.Info(msg)
	case types.ERR_WARNING:
		logger.Warning(msg)
	case types.ERR_OTHER:
		logger.Error(msg)
	case types.ERR_FATAL:
		logger.Fatal(msg)
	}
}

func (logger *FileLogger) Info(msg string) {
	logger.log(LOG_INFO, msg)
}

func (logger *FileLogger) Warning(msg string) {
	logger.log(LOG_WARNING, msg)
}

func (logger *FileLogger) Error(msg string) {
	logger.log(LOG_ERROR, msg)
}

func (logger *FileLogger) Fatal(msg string) {
	logger.log(LOG_FATAL, msg)
}

func (logger *FileLogger) Close() {
	logger.File.Close()
}

func (logger *FileLogger) getFilename(directory string, name string) string {
	pathString := path.Join(directory, name)
	return fmt.Sprintf("%s.log", pathString)
}

func (logger *FileLogger) log(level LogLevel, str string) {
	if level >= logger.Level && logger.Level != LOG_NOTSET {
		msg := fmt.Sprintf("[%s]: %s", level.String(), str)
		// To log file
		log.Println(msg)

		// To output
		fmt.Println(msg)
	}
}
