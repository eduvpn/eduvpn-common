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

type FileLogger struct {
	Level LogLevel
	File  *os.File
}

type LogLevel int8

const (
	// No level set, not allowed
	LogNotSet LogLevel = iota
	// Log debug, this message is not an error but is there for debugging
	LogDebug
	// Log info, this message is not an error but is there for additional information
	LogInfo
	// Log only to provide a warning, the app still functions
	LogWarning
	// Log to provide a generic error, the app still functions but some functionality might not work
	LogError
	// Log to provide a fatal error, the app cannot function correctly when such an error occurs
	LogFatal
)

func (e LogLevel) String() string {
	switch e {
	case LogNotSet:
		return "NOTSET"
	case LogDebug:
		return "DEBUG"
	case LogInfo:
		return "INFO"
	case LogWarning:
		return "WARNING"
	case LogError:
		return "ERROR"
	case LogFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

func (logger *FileLogger) Init(level LogLevel, directory string) error {
	errorMessage := "failed creating log"

	configDirErr := util.EnsureDirectory(directory)
	if configDirErr != nil {
		return types.NewWrappedError(errorMessage, configDirErr)
	}
	logFile, logOpenErr := os.OpenFile(
		logger.getFilename(directory),
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0o666,
	)
	if logOpenErr != nil {
		return types.NewWrappedError(errorMessage, logOpenErr)
	}
	multi := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multi)
	logger.File = logFile
	logger.Level = level
	return nil
}

func (logger *FileLogger) Inherit(label string, err error) {
	level := types.GetErrorLevel(err)

	msg := fmt.Sprintf("%s with err: %s", label, types.GetErrorTraceback(err))
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

func (logger *FileLogger) Debug(msg string, params ...interface{}) {
	logger.log(LogDebug, msg, params...)
}

func (logger *FileLogger) Info(msg string, params ...interface{}) {
	logger.log(LogInfo, msg, params...)
}

func (logger *FileLogger) Warning(msg string, params ...interface{}) {
	logger.log(LogWarning, msg, params...)
}

func (logger *FileLogger) Error(msg string, params ...interface{}) {
	logger.log(LogError, msg, params...)
}

func (logger *FileLogger) Fatal(msg string, params ...interface{}) {
	logger.log(LogFatal, msg, params...)
}

func (logger *FileLogger) Close() {
	logger.File.Close()
}

func (logger *FileLogger) getFilename(directory string) string {
	return path.Join(directory, "log")
}

func (logger *FileLogger) log(level LogLevel, msg string, params ...interface{}) {
	if level >= logger.Level && logger.Level != LogNotSet {
		formattedMsg := fmt.Sprintf(msg, params...)
		format := fmt.Sprintf("- Go - %s - %s", level.String(), formattedMsg)
		// To log file
		log.Println(format)
	}
}
