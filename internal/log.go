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
		return configDirErr
	}
	logFile, logOpenErr := os.OpenFile(logger.getFilename(directory, name), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
	if logOpenErr != nil {
		return logOpenErr
	}
	log.SetOutput(logFile)
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
