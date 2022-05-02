package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
)

type Config struct {
	Name      string
	Directory string
}

func (config *Config) Init(name string, directory string) {
	config.Name = name
	config.Directory = directory
}

func (config *Config) GetFilename() string {
	pathString := path.Join(config.Directory, config.Name)
	return fmt.Sprintf("%s.json", pathString)
}

func (config *Config) Save(readStruct interface{}) error {
	configDirErr := EnsureDirectory(config.Directory)
	if configDirErr != nil {
		return &ConfigSaveError{Err: configDirErr}
	}
	jsonString, marshalErr := json.Marshal(readStruct)
	if marshalErr != nil {
		return &ConfigSaveError{Err: marshalErr}
	}
	return ioutil.WriteFile(config.GetFilename(), jsonString, 0o644)
}

func (config *Config) Load(writeStruct interface{}) error {
	bytes, readErr := ioutil.ReadFile(config.GetFilename())
	if readErr != nil {
		return &ConfigLoadError{Err: readErr}
	}
	return json.Unmarshal(bytes, writeStruct)
}

type ConfigSaveError struct {
	Err error
}

func (e *ConfigSaveError) Error() string {
	return fmt.Sprintf("failed to save config with error: %v", e.Err)
}

type ConfigLoadError struct {
	Err error
}

func (e *ConfigLoadError) Error() string {
	return fmt.Sprintf("failed to load config with error: %v", e.Err)
}
