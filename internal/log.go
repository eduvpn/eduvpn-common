package internal

import (
	"fmt"
	"log"
	"os"
	"path"
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
	configDirErr := EnsureDirectory(directory)
	if configDirErr != nil {
		return &LogInitializeError{Name: name, Directory: directory, Err: configDirErr}
	}
	logFile, logOpenErr := os.OpenFile(logger.getFilename(directory, name), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
	if logOpenErr != nil {
		return &LogInitializeError{Name: name, Directory: directory, Err: logOpenErr}
	}
	log.SetOutput(logFile)
	logger.File = logFile
	logger.Level = level
	return nil
}

func (logger *FileLogger) getFilename(directory string, name string) string {
	pathString := path.Join(directory, name)
	return fmt.Sprintf("%s.log", pathString)
}

func (logger *FileLogger) Log(level LogLevel, str string) {
	if level >= logger.Level && logger.Level != LOG_NOTSET {
		log.Printf("[%s]: %s", level.String(), str)
	}
}

func (logger *FileLogger) Close() {
	logger.File.Close()
}

type LogInitializeError struct {
	Name      string
	Directory string
	Err       error
}

func (e *LogInitializeError) Error() string {
	return fmt.Sprintf("failed initializing logging with name: %s and directory: %s with error: %v", e.Name, e.Directory, e.Err)
}
