//go:build cgotesting

package main

import "testing"

func TestRegister(t *testing.T) {
	testRegister(t)
}

func TestServerList(t *testing.T) {
	testServerList(t)
}

func TestGetConfig(t *testing.T) {
	testGetConfig(t)
}

func TestLetsConnectDiscovery(t *testing.T) {
	testLetsConnectDiscovery(t)
}
