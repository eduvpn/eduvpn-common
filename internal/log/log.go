package log

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/eduvpn/eduvpn-common/types"
	"github.com/eduvpn/eduvpn-common/internal/util"
)

type FileLogger struct {
	Level LogLevel
	File  *os.File
}

type LogLevel int8

const (
	LOG_NOTSET LogLevel = iota
	LOG_INFO
	LOG_WARNING
	LOG_ERROR
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
	default:
		return "UNKNOWN"
	}
}

func (logger *FileLogger) Init(level LogLevel, name string, directory string) error {
	errorMessage := "failed creating log"

	configDirErr := util.EnsureDirectory(directory)
	if configDirErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: configDirErr}
	}
	logFile, logOpenErr := os.OpenFile(
		logger.getFilename(directory, name),
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0o666,
	)
	if logOpenErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: logOpenErr}
	}
	log.SetOutput(logFile)
	logger.File = logFile
	logger.Level = level
	return nil
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
