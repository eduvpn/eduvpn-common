// Package config implements functions for saving a struct to a file
// It then provides functions to later read it such that we can restore the same struct
package config

import (
	"encoding/json"
	"os"
	"path"

	"github.com/eduvpn/eduvpn-common/internal/config/v1"
	"github.com/eduvpn/eduvpn-common/internal/config/v2"
	"github.com/eduvpn/eduvpn-common/internal/discovery"
	"github.com/eduvpn/eduvpn-common/internal/log"
	"github.com/eduvpn/eduvpn-common/internal/util"
)

const stateFile = "state.json"

type Config struct {
	directory string
	V2        *v2.V2
}

func (c *Config) filename() string {
	return path.Join(c.directory, stateFile)
}

func (c *Config) Discovery() *discovery.Discovery {
	return &c.V2.Discovery
}

func (c *Config) Save() error {
	if err := util.EnsureDirectory(c.directory); err != nil {
		return err
	}

	join := Versioned{V2: c.V2}
	cfg, err := json.Marshal(join)
	if err != nil {
		return err
	}
	if err = os.WriteFile(c.filename(), cfg, 0o600); err != nil {
		return err
	}
	return nil
}

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

type Versioned struct {
	V1 *v1.V1 `json:"v1,omitempty"`
	V2 *v2.V2 `json:"v2,omitempty"`
}

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
