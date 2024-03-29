package util

import "testing"

func TestReplaceWAYF(t *testing.T) {
	// We expect url encoding but the spaces to be correctly replace with a + instead of a %20
	// And we expect that the return to and org_id are correctly replaced
	replaced := ReplaceWAYF(
		"@RETURN_TO@@ORG_ID@",
		"127.0.0.1:8000/&%$3#kM_-            ",
		"idp-test.nl.org/",
	)
	wantReplaced := "127.0.0.1%3A8000%2F%26%25%243%23kM_-++++++++++++idp-test.nl.org%2F"
	if replaced != wantReplaced {
		t.Fatalf("Got: %s, want: %s", replaced, wantReplaced)
	}

	// No RETURN_TO in template
	replaced = ReplaceWAYF("@ORG_ID@", "127.0.0.1:8000", "idp-test.nl.org/")
	wantReplaced = "127.0.0.1:8000"
	if replaced != wantReplaced {
		t.Fatalf("Got: %s, want: %s", replaced, wantReplaced)
	}

	// NO ORG_ID in template
	replaced = ReplaceWAYF("@RETURN_TO@", "127.0.0.1:8000", "idp-test.nl.org")
	wantReplaced = "127.0.0.1:8000"
	if replaced != wantReplaced {
		t.Fatalf("Got: %s, want: %s", replaced, wantReplaced)
	}

	// Template is empty
	replaced = ReplaceWAYF("", "127.0.0.1:8000", "idp-test.nl.org")
	wantReplaced = "127.0.0.1:8000"
	if replaced != wantReplaced {
		t.Fatalf("Got: %s, want: %s", replaced, wantReplaced)
	}

	// Template contains both @RETURN_TO@ and @ORG_ID@ but there is not enough to replace both
	replaced = ReplaceWAYF("@RETURN_TO@ORG_ID@", "127.0.0.1:8000", "idp-test.nl.org")
	wantReplaced = "127.0.0.1:8000"
	if replaced != wantReplaced {
		t.Fatalf("Got: %s, want: %s", replaced, wantReplaced)
	}
}
