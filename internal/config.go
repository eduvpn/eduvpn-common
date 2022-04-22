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
		return configDirErr
	}
	jsonString, marshalErr := json.Marshal(readStruct)
	if marshalErr != nil {
		return marshalErr
	}
	return ioutil.WriteFile(config.GetFilename(), jsonString, 0o644)
}

func (config *Config) Load(writeStruct interface{}) error {
	bytes, readErr := ioutil.ReadFile(config.GetFilename())
	if readErr != nil {
		return readErr
	}
	return json.Unmarshal(bytes, writeStruct)
}
