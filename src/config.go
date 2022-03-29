package eduvpn

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

func (eduvpn *VPNState) EnsureConfigDir() error {
	mkdirErr := os.MkdirAll(eduvpn.ConfigDirectory, os.ModePerm)
	if mkdirErr != nil {
		return mkdirErr
	}
	return nil
}

func (eduvpn *VPNState) GetConfigName() string {
	pathString := path.Join(eduvpn.ConfigDirectory, eduvpn.Name)
	return fmt.Sprintf("%s.json", pathString)
}

func (eduvpn *VPNState) WriteConfig() error {
	configDirErr := eduvpn.EnsureConfigDir()
	if configDirErr != nil {
		return configDirErr
	}
	jsonString, marshalErr := json.Marshal(eduvpn)
	if marshalErr != nil {
		return marshalErr
	}
	return ioutil.WriteFile(eduvpn.GetConfigName(), jsonString, 0o644)
}

func (eduvpn *VPNState) LoadConfig() error {
	bytes, readErr := ioutil.ReadFile(eduvpn.GetConfigName())
	if readErr != nil {
		return readErr
	}
	return json.Unmarshal(bytes, eduvpn)
}
