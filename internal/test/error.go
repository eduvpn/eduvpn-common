package test

import "testing"

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
