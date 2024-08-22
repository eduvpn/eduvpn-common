//go:build cgotesting

package main

import "testing"

func TestRegister(t *testing.T) {
	testRegister(t)
}

func TestServerList(t *testing.T) {
	testServerList(t)
}
