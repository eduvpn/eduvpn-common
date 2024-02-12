package test

import "testing"

// AssertError asserts an error by checking if the `Error()` strings are equal
func AssertError(t *testing.T, err error, wantErr string) {
	gv := ""
	if err != nil {
		gv = err.Error()
	}
	if wantErr != gv {
		if wantErr == "" {
			wantErr = "empty string"
		}
		t.Fatalf("Errors not equal, got: %v, want: %v", gv, wantErr)
	}
}
