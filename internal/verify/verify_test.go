package verify

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func Test_verifyWithKeys(t *testing.T) {
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

	var (
		verifyCreatePublicKeyError           *VerifyCreatePublicKeyError
		verifyInvalidSignatureAlgorithmError *VerifyInvalidSignatureAlgorithmError
		verifyWrongSigFilenameError          *VerifyWrongSigFilenameError
		verifyInvalidTrustedCommentError     *VerifyInvalidTrustedCommentError
		verifyInvalidSignatureFormatError    *VerifyInvalidSignatureFormatError
		verifyInvalidSignatureError          *VerifyInvalidSignatureError
		verifySigTimeEarlierError            *VerifySigTimeEarlierError
		verifyUnknownExpectedFilenameError   *VerifyUnknownExpectedFilenameError
		verifyUnknownKeyError                *VerifyUnknownKeyError
	)

	tests := []struct {
		expectedErr      interface{}
		testName         string
		signatureFile    string
		jsonFile         string
		expectedFileName string
		minSignTime      uint64
		allowedPks       []string
	}{
		{
			&verifyInvalidSignatureAlgorithmError,
			"pure",
			"server_list.json.pure.minisig",
			"server_list.json",
			"server_list.json",
			10,
			pk,
		},

		{
			nil,
			"valid server_list",
			"server_list.json.minisig",
			"server_list.json",
			"server_list.json",
			10,
			pk,
		},
		{
			nil,
			"TC no hashed",
			"server_list.json.tc_nohashed.minisig",
			"server_list.json",
			"server_list.json",
			10,
			pk,
		},
		{
			nil,
			"TC later time",
			"server_list.json.tc_latertime.minisig",
			"server_list.json",
			"server_list.json",
			10,
			pk,
		},
		{
			&verifyWrongSigFilenameError,
			"server_list TC file:organization_list",
			"server_list.json.tc_orglist.minisig",
			"server_list.json",
			"server_list.json",
			10,
			pk,
		},
		{
			&verifyWrongSigFilenameError,
			"organization_list as server_list",
			"organization_list.json.minisig",
			"organization_list.json",
			"server_list.json",
			10,
			pk,
		},
		{
			&verifyWrongSigFilenameError,
			"TC file:otherfile",
			"server_list.json.tc_otherfile.minisig",
			"server_list.json",
			"server_list.json",
			10,
			pk,
		},
		{
			&verifySigTimeEarlierError,
			"TC no file",
			"server_list.json.tc_nofile.minisig",
			"server_list.json",
			"server_list.json",
			10,
			pk,
		},
		{
			&verifySigTimeEarlierError,
			"TC no time",
			"server_list.json.tc_notime.minisig",
			"server_list.json",
			"server_list.json",
			10,
			pk,
		},
		{
			&verifySigTimeEarlierError,
			"TC empty time",
			"server_list.json.tc_emptytime.minisig",
			"server_list.json",
			"server_list.json",
			10,
			pk,
		},
		{
			&verifyInvalidSignatureFormatError,
			"TC empty file",
			"server_list.json.tc_emptyfile.minisig",
			"server_list.json",
			"server_list.json",
			10,
			pk,
		},
		{
			&verifyInvalidTrustedCommentError,
			"TC random",
			"server_list.json.tc_random.minisig",
			"server_list.json",
			"server_list.json",
			10,
			pk,
		},
		{
			nil,
			"large time",
			"server_list.json.large_time.minisig",
			"server_list.json",
			"server_list.json",
			43e8,
			pk,
		},
		{
			nil,
			"lower min time",
			"server_list.json.minisig",
			"server_list.json",
			"server_list.json",
			5,
			pk,
		},
		{
			&verifySigTimeEarlierError,
			"higher min time",
			"server_list.json.minisig",
			"server_list.json",
			"server_list.json",
			11,
			pk,
		},

		{
			nil,
			"valid organization_list",
			"organization_list.json.minisig",
			"organization_list.json",
			"organization_list.json",
			10,
			pk,
		},
		{
			&verifyWrongSigFilenameError,
			"organization_list TC file:server_list",
			"organization_list.json.tc_servlist.minisig",
			"organization_list.json",
			"organization_list.json",
			10,
			pk,
		},
		{
			&verifyWrongSigFilenameError,
			"server_list as organization_list",
			"server_list.json.minisig",
			"server_list.json",
			"organization_list.json",
			10,
			pk,
		},

		{
			&verifyUnknownExpectedFilenameError,
			"valid other_list",
			"other_list.json.minisig",
			"other_list.json",
			"other_list.json",
			10,
			pk,
		},
		{
			&verifyWrongSigFilenameError,
			"other_list as server_list",
			"other_list.json.minisig",
			"other_list.json",
			"server_list.json",
			10,
			pk,
		},

		{
			&verifyInvalidSignatureFormatError,
			"invalid signature file",
			"random.txt",
			"server_list.json",
			"server_list.json",
			10,
			pk,
		},
		{
			&verifyInvalidSignatureFormatError,
			"empty signature file",
			"empty",
			"server_list.json",
			"server_list.json",
			10,
			pk,
		},

		{
			&verifyUnknownKeyError,
			"wrong key",
			"server_list.json.wrong_key.minisig",
			"server_list.json",
			"server_list.json",
			10,
			pk,
		},

		{
			&verifyInvalidSignatureAlgorithmError,
			"forged pure signature",
			"server_list.json.forged_pure.minisig",
			"server_list.json.blake2b",
			"server_list.json",
			10,
			pk,
		},
		{
			&verifyInvalidSignatureError,
			"forged key ID",
			"server_list.json.forged_keyid.minisig",
			"server_list.json",
			"server_list.json",
			10,
			pk,
		},

		{
			&verifyUnknownKeyError,
			"no allowed keys",
			"server_list.json.minisig",
			"server_list.json",
			"server_list.json",
			10,
			[]string{},
		},
		{
			nil,
			"multiple allowed keys 1",
			"server_list.json.minisig",
			"server_list.json",
			"server_list.json",
			10,
			[]string{
				pk[0], "RWSf0PYToIUJmDlsz21YOXvgQzHj9NSdyJUqEY5ZdfS9GepeXt3+JJRZ",
			},
		},
		{
			nil,
			"multiple allowed keys 2",
			"server_list.json.minisig",
			"server_list.json",
			"server_list.json",
			10,
			[]string{
				"RWSf0PYToIUJmDlsz21YOXvgQzHj9NSdyJUqEY5ZdfS9GepeXt3+JJRZ", pk[0],
			},
		},
		{
			&verifyCreatePublicKeyError,
			"invalid allowed key",
			"server_list.json.minisig",
			"server_list.json",
			"server_list.json",
			10,
			[]string{"AAA"},
		},
	}

	// Cache file contents in map, mapping file names to contents
	files := map[string][]byte{}
	loadFile := func(name string) {
		_, loaded := files[name]
		if !loaded {
			content, err := ioutil.ReadFile("test_data/" + name)
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

	forcePrehash := true
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()
			valid, err := verifyWithKeys(string(files[tt.signatureFile]), files[tt.jsonFile],
				tt.expectedFileName, tt.minSignTime, tt.allowedPks, forcePrehash)
			compareResults(t, valid, err, tt.expectedErr, func() string {
				return fmt.Sprintf(
					"verifyWithKeys(%q, %q, %q, %v, %v, %t)",
					tt.signatureFile,
					tt.jsonFile,
					tt.expectedFileName,
					tt.minSignTime,
					tt.allowedPks,
					forcePrehash,
				)
			})
		})
	}
}

// compareResults compares returned ret, err from a verify function with expected error code expected.
// callStr is called to get the formatted parameters passed to the function.
func compareResults(
	t *testing.T,
	ret bool,
	err error,
	expectedErr interface{},
	callStr func() string,
) {
	// different error returned
	if expectedErr != nil && !errors.As(err, expectedErr) {
		t.Errorf("%v\nerror %T = %v, wantErr %T", callStr(), err, err, expectedErr)
		return
	}
	// different boolean returned
	expectedBool := expectedErr == nil
	if ret != expectedBool {
		t.Errorf("%v\n= %v, want %v", callStr(), ret, expectedBool)
	}
}
