package eduvpn_discovery

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
)

func Test_verifyWithKeys(t *testing.T) {
	var err error

	var pk []string
	{
		file, err := os.Open("test_data/public.key")
		if err != nil {
			panic(err)
		}
		defer file.Close()

		// Get last line (key string) from file
		scanner := bufio.NewScanner(file)
		for i := 0; i < 2; i++ {
			if !scanner.Scan() {
				panic(scanner.Err())
			}
		}
		pk = []string{scanner.Text()}
	}

	tests := []struct {
		result           detailedVerifyErrorCode
		testName         string
		signatureFile    string
		jsonFile         string
		expectedFileName string
		minSignTime      uint64
		allowedPks       []string
	}{
		{errInvalidSignatureAlgorithm, "pure", "server_list.json.pure.minisig", "server_list.json", "server_list.json", 10, pk},

		{ok, "valid server_list", "server_list.json.minisig", "server_list.json", "server_list.json", 10, pk},
		{ok, "TC no hashed", "server_list.json.tc_nohashed.minisig", "server_list.json", "server_list.json", 10, pk},
		{ok, "TC later time", "server_list.json.tc_latertime.minisig", "server_list.json", "server_list.json", 10, pk},
		{errWrongFileName, "server_list TC file:organization_list", "server_list.json.tc_orglist.minisig", "server_list.json", "server_list.json", 10, pk},
		{errWrongFileName, "organization_list as server_list", "organization_list.json.minisig", "organization_list.json", "server_list.json", 10, pk},
		{errWrongFileName, "TC file:otherfile", "server_list.json.tc_otherfile.minisig", "server_list.json", "server_list.json", 10, pk},
		{errInvalidTrustedComment, "TC no file", "server_list.json.tc_nofile.minisig", "server_list.json", "server_list.json", 10, pk},
		{errInvalidTrustedComment, "TC no time", "server_list.json.tc_notime.minisig", "server_list.json", "server_list.json", 10, pk},
		{errAny, "TC empty time", "server_list.json.tc_emptytime.minisig", "server_list.json", "server_list.json", 10, pk},
		{errAny, "TC empty file", "server_list.json.tc_emptyfile.minisig", "server_list.json", "server_list.json", 10, pk},
		{errInvalidTrustedComment, "TC random", "server_list.json.tc_random.minisig", "server_list.json", "server_list.json", 10, pk},
		{ok, "large time", "server_list.json.large_time.minisig", "server_list.json", "server_list.json", 43e8, pk},
		{ok, "lower min time", "server_list.json.minisig", "server_list.json", "server_list.json", 5, pk},
		{errTooOld, "higher min time", "server_list.json.minisig", "server_list.json", "server_list.json", 11, pk},

		{ok, "valid organization_list", "organization_list.json.minisig", "organization_list.json", "organization_list.json", 10, pk},
		{errWrongFileName, "organization_list TC file:server_list", "organization_list.json.tc_servlist.minisig", "organization_list.json", "organization_list.json", 10, pk},
		{errWrongFileName, "server_list as organization_list", "server_list.json.minisig", "server_list.json", "organization_list.json", 10, pk},

		{errUnknownExpectedFileName, "valid other_list", "other_list.json.minisig", "other_list.json", "other_list.json", 10, pk},
		{errWrongFileName, "other_list as server_list", "other_list.json.minisig", "other_list.json", "server_list.json", 10, pk},

		{errInvalidSignatureFormat, "invalid signature file", "random.txt", "server_list.json", "server_list.json", 10, pk},
		{errInvalidSignatureFormat, "empty signature file", "empty", "server_list.json", "server_list.json", 10, pk},

		{errWrongKey, "wrong key", "server_list.json.wrong_key.minisig", "server_list.json", "server_list.json", 10, pk},

		{errInvalidSignatureAlgorithm, "forged pure signature", "server_list.json.forged_pure.minisig", "server_list.json.blake2b", "server_list.json", 10, pk},
		{errInvalidSignature, "forged key ID", "server_list.json.forged_keyid.minisig", "server_list.json", "server_list.json", 10, pk},

		{errWrongKey, "no allowed keys", "server_list.json.minisig", "server_list.json", "server_list.json", 10, []string{}},
		{ok, "multiple allowed keys 1", "server_list.json.minisig", "server_list.json", "server_list.json", 10, []string{
			pk[0], "RWSf0PYToIUJmDlsz21YOXvgQzHj9NSdyJUqEY5ZdfS9GepeXt3+JJRZ",
		}},
		{ok, "multiple allowed keys 2", "server_list.json.minisig", "server_list.json", "server_list.json", 10, []string{
			"RWSf0PYToIUJmDlsz21YOXvgQzHj9NSdyJUqEY5ZdfS9GepeXt3+JJRZ", pk[0],
		}},
		{errInvalidPublicKey, "invalid allowed key", "server_list.json.minisig", "server_list.json", "server_list.json", 10, []string{"AAA"}},
	}

	// Cache file contents in map, mapping file names to contents
	files := map[string][]byte{}
	loadFile := func(name string) {
		content, loaded := files[name]
		if !loaded {
			content, err = ioutil.ReadFile("test_data/" + name)
			if err != nil {
				panic(err)
			}
			files[name] = content
		}
	}
	for _, test := range tests {
		loadFile(test.signatureFile)
		loadFile(test.jsonFile)
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()
			valid, err := verifyWithKeys(string(files[tt.signatureFile]), files[tt.jsonFile],
				tt.expectedFileName, tt.minSignTime, tt.allowedPks)
			compareResults(t, valid, err, int(tt.result), func() string {
				return fmt.Sprintf("verifyWithKeys(%q, %q, %q, %v, %v)",
					tt.signatureFile, tt.jsonFile, tt.expectedFileName, tt.minSignTime, tt.allowedPks)
			})
		})
	}
}

const (
	ok     = -1 // Test should not give an error.
	errAny = -2 // Test should give any error (specific error is an implementation detail).
)

// compareResults compares returned ret, err from a verify function with expected error code expected.
// callStr is called to get the formatted parameters passed to the function.
func compareResults(t *testing.T, ret bool, err error, expected int, callStr func() string) {
	getCode := func(err error) int {
		switch e := err.(type) {
		case detailedVerifyError:
			return int(e.Code)
		case VerifyError:
			return int(e.Code)
		}
		panic(nil)
	}

	if (err == nil) != (expected == ok) || err != nil && expected != errAny && getCode(err) != expected {
		var errMsg string
		if err != nil {
			errMsg = fmt.Sprintf("%v %v (cause %v)", getCode(err), err, errors.Unwrap(err))
		} else {
			errMsg = "<ok>"
		}

		var wantErrCode string
		switch expected {
		case ok:
			wantErrCode = "<ok>"
		case errAny:
			wantErrCode = "<any>"
		default:
			wantErrCode = strconv.Itoa(expected)
		}

		t.Errorf("%v\nerror = %v, wantErr %v", callStr(), errMsg, wantErrCode)
		return
	}
	if ret != (expected == ok) {
		t.Errorf("%v\n= %v, want %v", callStr(), ret, expected == ok)
	}
}
