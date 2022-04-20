package eduvpn

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

func (eduvpn *VPNState) getLogFilename() string {
	pathString := path.Join(eduvpn.ConfigDirectory, eduvpn.Name)
	return fmt.Sprintf("%s.log", pathString)
}

func (eduvpn *VPNState) InitLog(level LogLevel) error {
	configDirErr := eduvpn.EnsureConfigDir()
	if configDirErr != nil {
		return configDirErr
	}
	logFile, logOpenErr := os.OpenFile(eduvpn.getLogFilename(), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
	if logOpenErr != nil {
		return logOpenErr
	}
	log.SetOutput(logFile)
	eduvpn.LogFile = FileLogger{Level: level, File: logFile}
	return nil
}

func (eduvpn *VPNState) Log(level LogLevel, str string) {
	if level >= eduvpn.LogFile.Level && eduvpn.LogFile.Level != LOG_NOTSET {
		log.Printf("[%s]: %s", level.String(), str)
	}
}

func (eduvpn *VPNState) CloseLog() {
	eduvpn.LogFile.File.Close()
}
