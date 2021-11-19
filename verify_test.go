package eduvpn_verify

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
)

const (
	ok     VerifyErrCode = -1
	errAny               = -2
)

func compareResults(t *testing.T, ret bool, err error, expected VerifyErrCode, call func() string) {
	if (err == nil) != (expected == ok) || err != nil && expected != errAny && err.(VerifyError).Code != expected {
		var errMsg string
		if err != nil {
			errMsg = fmt.Sprintf("%v %v (cause %v)", err.(VerifyError).Code, err, errors.Unwrap(err))
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
			wantErrCode = strconv.Itoa(int(expected))
		}

		t.Errorf("%v\nerror = %v, wantErr %v", call(), errMsg, wantErrCode)
		return
	}
	if ret != (expected == ok) {
		t.Errorf("%v\n= %v, want %v", call(), ret, expected == ok)
	}
}

func Test_verifyWithKeys(t *testing.T) {
	var err error

	var pk []string
	{
		file, err := os.Open("test_data/dummy/public.key")
		if err != nil {
			panic(err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for i := 0; i < 2; i++ {
			if !scanner.Scan() {
				panic(scanner.Err())
			}
		}
		pk = []string{scanner.Text()}
	}

	tests := []struct {
		result           VerifyErrCode
		testName         string
		signatureFile    string
		jsonFile         string
		expectedFileName string
		minSignTime      uint64
		allowedPks       []string
	}{
		{ErrInvalidSignatureAlgorithm, "pure", "server_list.json.pure.minisig", "server_list.json", "server_list.json", 10, pk},

		{ok, "valid server_list", "server_list.json.minisig", "server_list.json", "server_list.json", 10, pk},
		{ok, "TC timestamp:", "server_list.json.tc_timestamp.minisig", "server_list.json", "server_list.json", 10, pk},
		{ok, "TC no hashed", "server_list.json.tc_nohashed.minisig", "server_list.json", "server_list.json", 10, pk},
		{ok, "TC later time", "server_list.json.tc_latertime.minisig", "server_list.json", "server_list.json", 10, pk},
		{ErrWrongFileName, "server_list TC file:organization_list", "server_list.json.tc_orglist.minisig", "server_list.json", "server_list.json", 10, pk},
		{ErrWrongFileName, "organization_list as server_list", "organization_list.json.minisig", "organization_list.json", "server_list.json", 10, pk},
		{ErrWrongFileContent, "organization_list as server_list TC file:server_list", "organization_list.json.tc_servlist.minisig", "organization_list.json", "server_list.json", 10, pk},
		{ErrWrongFileName, "TC file:otherfile", "server_list.json.tc_otherfile.minisig", "server_list.json", "server_list.json", 10, pk},
		{ErrInvalidTrustedComment, "TC no file", "server_list.json.tc_nofile.minisig", "server_list.json", "server_list.json", 10, pk},
		{ErrInvalidTrustedComment, "TC no time", "server_list.json.tc_notime.minisig", "server_list.json", "server_list.json", 10, pk},
		{errAny, "TC empty time", "server_list.json.tc_emptytime.minisig", "server_list.json", "server_list.json", 10, pk},
		{errAny, "TC empty file", "server_list.json.tc_emptyfile.minisig", "server_list.json", "server_list.json", 10, pk},
		{ErrWrongFileContent, "TC earlier time", "server_list.json.tc_earliertime.minisig", "server_list.json", "server_list.json", 9, pk},
		{ErrInvalidTrustedComment, "TC random", "server_list.json.tc_random.minisig", "server_list.json", "server_list.json", 10, pk},
		{ok, "large time", "server_list.json.large_time.minisig", "server_list-large_time.json", "server_list.json", 43e8, pk},
		{ErrWrongFileContent, "no version", "server_list.json.no_version.minisig", "server_list-no_version.json", "server_list.json", 10, pk},
		{ok, "lower min time", "server_list.json.minisig", "server_list.json", "server_list.json", 5, pk},
		{ErrTooOld, "higher min time", "server_list.json.minisig", "server_list.json", "server_list.json", 11, pk},
		{ErrTooOld, "higher min time < signature time", "server_list.json.tc_latertime.minisig", "server_list.json", "server_list.json", 15, pk},

		{ok, "valid organization_list", "organization_list.json.minisig", "organization_list.json", "organization_list.json", 10, pk},
		{ErrWrongFileName, "organization_list TC file:server_list", "organization_list.json.tc_servlist.minisig", "organization_list.json", "organization_list.json", 10, pk},
		{ErrWrongFileName, "server_list as organization_list", "server_list.json.minisig", "server_list.json", "organization_list.json", 10, pk},
		{ErrWrongFileContent, "server_list as organization_list TC file:organization_list", "server_list.json.tc_orglist.minisig", "server_list.json", "organization_list.json", 10, pk},

		{ErrUnknownExpectedFileName, "valid other_list", "other_list.json.minisig", "other_list.json", "other_list.json", 10, pk},
		{ErrWrongFileName, "JSON other_list as server_list", "other_list.json.minisig", "other_list.json", "server_list.json", 10, pk},
		{ErrWrongFileContent, "JSON other_list as server_list TC file:server_list", "other_list.json.tc_servlist.minisig", "other_list.json", "server_list.json", 10, pk},
		{ErrWrongFileContent, "JSON no list", "no_list.json.minisig", "no_list.json", "server_list.json", 10, pk},
		{ErrWrongFileContent, "non-JSON file", "random.txt.minisig", "random.txt", "server_list.json", 10, pk},
		{ErrWrongFileContent, "empty file", "empty.minisig", "empty", "server_list.json", 10, pk},

		{ErrWrongFileContent, "JSON wrong type 1", "wrong_type1.json.minisig", "wrong_type1.json", "server_list.json", 10, pk},
		{ErrWrongFileContent, "JSON wrong type 2", "wrong_type2.json.minisig", "wrong_type2.json", "server_list.json", 10, pk},
		{ErrWrongFileContent, "JSON wrong type 3", "wrong_type3.json.minisig", "wrong_type3.json", "server_list.json", 10, pk},

		{ErrInvalidSignatureFormat, "invalid signature file", "random.txt", "server_list.json", "server_list.json", 10, pk},
		{ErrInvalidSignatureFormat, "empty signature file", "empty", "server_list.json", "server_list.json", 10, pk},

		{ErrWrongKey, "wrong key", "server_list.json.wrong_key.minisig", "server_list.json", "server_list.json", 10, pk},

		{ErrInvalidSignatureAlgorithm, "forged pure signature", "server_list.json.forged_pure.minisig", "server_list.json.blake2b", "server_list.json", 10, pk},
		{ErrInvalidSignature, "forged key ID", "server_list.json.forged_keyid.minisig", "server_list.json", "server_list.json", 10, pk},

		{ErrWrongKey, "no allowed keys", "server_list.json.minisig", "server_list.json", "server_list.json", 10, []string{}},
		{ok, "multiple allowed keys 1", "server_list.json.minisig", "server_list.json", "server_list.json", 10, []string{
			pk[0], "RWSf0PYToIUJmDlsz21YOXvgQzHj9NSdyJUqEY5ZdfS9GepeXt3+JJRZ",
		}},
		{ok, "multiple allowed keys 2", "server_list.json.minisig", "server_list.json", "server_list.json", 10, []string{
			"RWSf0PYToIUJmDlsz21YOXvgQzHj9NSdyJUqEY5ZdfS9GepeXt3+JJRZ", pk[0],
		}},
		{ErrInvalidPublicKey, "invalid allowed key", "server_list.json.minisig", "server_list.json", "server_list.json", 10, []string{"AAA"}},
	}

	files := map[string][]byte{}
	for _, test := range tests {
		signature, loaded := files[test.signatureFile]
		if !loaded {
			signature, err = ioutil.ReadFile("test_data/dummy/" + test.signatureFile)
			if err != nil {
				panic(err)
			}
			files[test.signatureFile] = signature
		}

		json, loaded := files[test.jsonFile]
		if !loaded {
			json, err = ioutil.ReadFile("test_data/dummy/" + test.jsonFile)
			if err != nil {
				panic(err)
			}
			files[test.jsonFile] = json
		}
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()
			valid, err := verifyWithKeys(string(files[tt.signatureFile]), files[tt.jsonFile],
				tt.expectedFileName, tt.minSignTime, tt.allowedPks)
			compareResults(t, valid, err, tt.result, func() string {
				return fmt.Sprintf("verifyWithKeys(%q, %q, %q, %v, %v)",
					tt.signatureFile, tt.jsonFile, tt.expectedFileName, tt.minSignTime, tt.allowedPks)
			})
		})
	}
}

func Test_Verify(t *testing.T) {
	var err error
	tests := []struct {
		result           VerifyErrCode
		testName         string
		signatureFile    string
		jsonFile         string
		expectedFileName string
		minSignTime      uint64
	}{
		//TODO tests with real valid *prehashed* signatures
		{ErrInvalidSignatureAlgorithm, "pure server_list", "server_list-1.json.pure.minisig", "server_list-1.json", "server_list.json", 1636532223},
		{ErrInvalidSignatureAlgorithm, "pure organization_list", "organization_list-1.json.pure.minisig", "organization_list-1.json", "organization_list.json", 1636532223},
		{ErrWrongKey, "wrong key", "../dummy/server_list.json.minisig", "../dummy/server_list.json", "server_list.json", 10},
	}

	files := map[string][]byte{}
	for _, test := range tests {
		signature, loaded := files[test.signatureFile]
		if !loaded {
			signature, err = ioutil.ReadFile("test_data/real/" + test.signatureFile)
			if err != nil {
				panic(err)
			}
			files[test.signatureFile] = signature
		}

		json, loaded := files[test.jsonFile]
		if !loaded {
			json, err = ioutil.ReadFile("test_data/real/" + test.jsonFile)
			if err != nil {
				panic(err)
			}
			files[test.jsonFile] = json
		}
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()
			valid, err := Verify(string(files[tt.signatureFile]), files[tt.jsonFile], tt.expectedFileName, tt.minSignTime)
			compareResults(t, valid, err, tt.result, func() string {
				return fmt.Sprintf("Verify(%q, %q, %q, %v)",
					tt.signatureFile, tt.jsonFile, tt.expectedFileName, tt.minSignTime)
			})
		})
	}
}
