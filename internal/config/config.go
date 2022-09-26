package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/eduvpn/eduvpn-common/types"
	"github.com/eduvpn/eduvpn-common/internal/util"
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
	errorMessage := "failed saving configuration"
	configDirErr := util.EnsureDirectory(config.Directory)
	if configDirErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: configDirErr}
	}
	jsonString, marshalErr := json.Marshal(readStruct)
	if marshalErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: marshalErr}
	}
	return ioutil.WriteFile(config.GetFilename(), jsonString, 0o600)
}

func (config *Config) Load(writeStruct interface{}) error {
	bytes, readErr := ioutil.ReadFile(config.GetFilename())
	if readErr != nil {
		return &types.WrappedErrorMessage{Message: "failed loading configuration", Err: readErr}
	}
	return json.Unmarshal(bytes, writeStruct)
}
