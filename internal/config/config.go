// Package config implements functions for saving a struct to a file
// It then provides functions to later read it such that we can restore the same struct
package config

import (
	"encoding/json"
	"os"
	"path"

	"codeberg.org/eduVPN/eduvpn-common/internal/config/atomicfile"
	"codeberg.org/eduVPN/eduvpn-common/internal/config/v1"
	"codeberg.org/eduVPN/eduvpn-common/internal/config/v2"
	"codeberg.org/eduVPN/eduvpn-common/internal/discovery"
	"codeberg.org/eduVPN/eduvpn-common/internal/log"
	"codeberg.org/eduVPN/eduvpn-common/internal/util"
)

const stateFile = "state.json"

// Config represents the config state file
type Config struct {
	directory string
	// V2 indicates we are version 2
	V2 *v2.V2
}

func (c *Config) filename() string {
	return path.Join(c.directory, stateFile)
}

// Discovery gets the discovery list from the state file
func (c *Config) Discovery() *discovery.Discovery {
	return &c.V2.Discovery
}

// HasSecureInternet returns whether or not the configuration has a secure internet server
func (c *Config) HasSecureInternet() bool {
	return c.V2.HasSecureInternet()
}

// Save saves the state file to disk
func (c *Config) Save() error {
	if err := util.EnsureDirectory(c.directory); err != nil {
		return err
	}

	join := Versioned{V2: c.V2}
	cfg, err := json.Marshal(join)
	if err != nil {
		return err
	}
	if err = atomicfile.WriteFile(c.filename(), cfg, 0o600); err != nil {
		return err
	}
	return nil
}

// Load loads the state file from disk
func (c *Config) Load() error {
	bts, err := os.ReadFile(c.filename())
	if err != nil {
		return err
	}
	var buf Versioned
	if err = json.Unmarshal(bts, &buf); err != nil {
		return err
	}
	if buf.V2 != nil {
		c.V2 = buf.V2
		return nil
	}
	if buf.V1 != nil {
		c.V2 = v2.FromV1(buf.V1)
	}
	return nil
}

// Versioned is the final top-level state file that is written to disk
type Versioned struct {
	// V1 is the version 1 state file that is no longer used but converted from
	V1 *v1.V1 `json:"v1,omitempty"`
	// V2 is the version 2 state file
	V2 *v2.V2 `json:"v2,omitempty"`
}

// NewFromDirectory creates a new config struct from a directory
// It does this by loading the JSON file from disk
func NewFromDirectory(dir string) *Config {
	cfg := Config{
		directory: dir,
	}
	err := cfg.Load()
	if err != nil {
		log.Logger.Debugf("failed to load configuration: %v", err)
	}
	if cfg.V2 == nil {
		cfg.V2 = &v2.V2{}
	}
	return &cfg
}
