package levenshtein

import "testing"

func TestLevenshtein(t *testing.T) {
	cases := []struct {
		a     string
		b     string
		score int
	}{
		{
			a:     "foo",
			b:     "foo",
			score: 0,
		},
		{
			a:     "foo",
			b:     "foo",
			score: 0,
		},
		{
			a:     "foo",
			b:     "foosd",
			score: 2,
		},
		{
			a:     "foo",
			b:     "bla",
			score: 3,
		},
		{
			a:     "foo",
			b:     "",
			score: 3,
		},
		{
			a:     "",
			b:     "foo",
			score: 3,
		},
	}
	for i, c := range cases {
		g := levenshtein(c.a, c.b)
		if g != c.score {
			t.Fatalf("case %d not equal, got: %d, want: %d", i, g, c.score)
		}
	}
}

func TestAdjusted(t *testing.T) {
	cases := []struct {
		a     string
		b     string
		score int
	}{
		{
			a:     "foo",
			b:     "foo",
			score: 0,
		},
		{
			a:     "foo",
			b:     "foo",
			score: 0,
		},
		{
			a:     "foo",
			b:     "foosd",
			score: 2,
		},
		{
			a:     "foo",
			b:     "bla",
			score: -1,
		},
		{
			a:     "bla foo",
			b:     "bla",
			score: -1,
		},
		{
			a:     "foo",
			b:     "",
			score: -1,
		},
		{
			a:     "",
			b:     "foo",
			score: 3,
		},
	}
	for i, c := range cases {
		g := adjusted(c.a, c.b)
		if g != c.score {
			t.Fatalf("case %d not equal, got: %d, want: %d", i, g, c.score)
		}
	}
}

func TestDiscoveryScore(t *testing.T) {
	cases := []struct {
		q     string
		disp  map[string]string
		keys  map[string]string
		score int
	}{
		{
			q: "test",
			disp: map[string]string{
				"en": "test",
				"de": "test",
			},
			keys: map[string]string{
				"en": "testing",
				"de": "testing",
			},
			score: 0,
		},
		{
			q: "test",
			disp: map[string]string{
				"en": "testing",
				"de": "testing",
			},
			keys: map[string]string{
				"en": "test",
				"de": "test",
			},
			score: 2,
		},
		{
			q: "foo",
			disp: map[string]string{
				"en": "test",
				"de": "testing",
			},
			keys: map[string]string{
				"en": "foo",
				"de": "foo",
			},
			score: 2,
		},
		{
			q: "fox",
			disp: map[string]string{
				"en": "test",
				"de": "testing",
			},
			keys: map[string]string{
				"en": "foo",
				"de": "foo",
			},
			score: -2,
		},
	}

	for i, c := range cases {
		g := DiscoveryScore(c.q, c.disp, c.keys)
		if g != c.score {
			t.Fatalf("case %d not equal, got: %d, want: %d", i, g, c.score)
		}
	}
}

func TestRemoveDiacritics(t *testing.T) {
	cases := []struct {
		input string
		want  string
		e     error
	}{
		{
			input: "foobar",
			want:  "foobar",
			e:     nil,
		},
		{
			input: "fòóbår",
			want:  "foobar",
			e:     nil,
		},
		{
			input: "GÉANT",
			want:  "GEANT",
			e:     nil,
		},
	}

	for _, c := range cases {
		result, e := removeDiacritics(c.input)
		if result != c.want || e != c.e {
			t.Fatalf("Result: %s, %v Want: %s, %v", result, e, c.want, c.e)
		}
	}
}
