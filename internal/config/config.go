// Package config implements functions for saving a struct to a file
// It then provides functions to later read it such that we can restore the same struct
package config

import (
	"encoding/json"
	"os"
	"path"

	"github.com/eduvpn/eduvpn-common/internal/util"
	"github.com/go-errors/errors"
)

// Config represents a configuration that saves the client's struct as JSON.
type Config struct {
	// Directory represents the path to where the data is saved
	Directory string

	// Name defines the name of file excluding the .json extension
	Name string
}

type ConfigFormat struct {
	Data interface{} `json:"v1"`
}

// Init initializes the configuration using the provided directory and name.
func (c *Config) Init(directory string, name string) {
	c.Directory = directory
	c.Name = name
}

// filename returns the filename of the configuration as a full path.
func (c *Config) filename() string {
	return path.Join(c.Directory, c.Name) + ".json"
}

// Save saves a structure 'readStruct' to the configuration
// If it was unsuccessful, an error is returned.
func (c *Config) Save(readStruct interface{}) error {
	if err := util.EnsureDirectory(c.Directory); err != nil {
		return err
	}
	cf := &ConfigFormat{Data: readStruct}
	cfg, err := json.Marshal(cf)
	if err != nil {
		return errors.WrapPrefix(err, "json.Marshal failed", 0)
	}
	if err = os.WriteFile(c.filename(), cfg, 0o600); err != nil {
		return errors.WrapPrefix(err, "os.WriteFile failed", 0)
	}
	return nil
}

// Load loads the configuration and writes the structure to 'writeStruct'
// If it was unsuccessful, an error is returned.
func (c *Config) Load(writeStruct interface{}) error {
	bts, err := os.ReadFile(c.filename())
	if err != nil {
		return errors.WrapPrefix(err, "failed loading configuration", 0)
	}
	cf := ConfigFormat{Data: writeStruct}
	if err = json.Unmarshal(bts, &cf); err != nil {
		return errors.WrapPrefix(err, "json.Unmarshal failed", 0)
	}
	return nil
}
