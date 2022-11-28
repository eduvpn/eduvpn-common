// Package config implements functions for saving a struct to a file
// It then provides functions to later read it such that we can restore the same struct
package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/eduvpn/eduvpn-common/internal/util"
	"github.com/eduvpn/eduvpn-common/types"
)

// Config represents a configuration that saves the client's struct as JSON.
type Config struct {
	// Directory represents the path to where the data is saved
	Directory string

	// Name defines the name of file excluding the .json extension
	Name string
}

// Init initializes the configuration using the provided directory and name.
func (config *Config) Init(directory string, name string) {
	config.Directory = directory
	config.Name = name
}

// filename returns the filename of the configuration as a full path.
func (config *Config) filename() string {
	pathString := path.Join(config.Directory, config.Name)
	return fmt.Sprintf("%s.json", pathString)
}

// Save saves a structure 'readStruct' to the configuration
// If it was unusuccessful, an an error is returned.
func (config *Config) Save(readStruct interface{}) error {
	errorMessage := "failed saving configuration"
	configDirErr := util.EnsureDirectory(config.Directory)
	if configDirErr != nil {
		return types.NewWrappedError(errorMessage, configDirErr)
	}
	jsonString, marshalErr := json.Marshal(readStruct)
	if marshalErr != nil {
		return types.NewWrappedError(errorMessage, marshalErr)
	}
	return ioutil.WriteFile(config.filename(), jsonString, 0o600)
}

// Load loads the configuration and writes the structure to 'writeStruct'
// If it was unsuccessful, an error is returned.
func (config *Config) Load(writeStruct interface{}) error {
	bytes, readErr := ioutil.ReadFile(config.filename())
	if readErr != nil {
		return types.NewWrappedError("failed loading configuration", readErr)
	}
	return json.Unmarshal(bytes, writeStruct)
}
