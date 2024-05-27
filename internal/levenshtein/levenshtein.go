package levenshtein

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// min returns the min of a and b
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// levenshtein is an algorithm that returns the "distance" between two strings
// the distance for hello and helloxd is 2 because it takes two inserts to go from hello to helloxd
// the distance between hello and hello is 0 because the strings are equal
// apart from insertions, the levenshtein algorithm also takes substitutions and deletions into account
// levenshtein implementation from https://en.wikipedia.org/wiki/Levenshtein_distance#Iterative_with_two_matrix_rows
func levenshtein(os, ot string) int {
	n := utf8.RuneCountInString(os)
	m := utf8.RuneCountInString(ot)
	s := []rune(os)
	t := []rune(ot)
	v0 := make([]int, m+1)
	v1 := make([]int, m+1)
	for i := 0; i <= m; i++ {
		v0[i] = i
	}

	for i := 0; i < n; i++ {
		v1[0] = i + 1
		for j := 0; j < m; j++ {
			// calculate deletion cost,
			// insertion cost and
			// substitution cost
			dc := v0[j+1] + 1
			ic := v1[j] + 1
			var sc int
			if s[i] == t[j] {
				sc = v0[j]
			} else {
				sc = v0[j] + 1
			}
			// take the min of all the costs
			v1[j+1] = min(min(dc, ic), sc)
		}
		v0, v1 = v1, v0
	}
	return v0[m]
}

// adjusted creates and adjusted version of the levenshtein algorithm
// where it filters entries where one of the words in the substr is not contained in `full`
// for these a score of -1 returned
// for all others it is the normal levenshtein distance
func adjusted(substr, full string) int {
	sSub := strings.Split(substr, " ")
	for _, vSub := range sSub {
		if !strings.Contains(full, vSub) {
			return -1
		}
	}
	return levenshtein(substr, full)
}

// KeywordPenalty is the penalty for matching on keywords instead of display names
const KeywordPenalty = 2

// DiscoveryScore computes the score of a discovery entry with the given search query
// a negative score means exclude the entry from the results
func DiscoveryScore(search string, displays map[string]string, keywords map[string]string) int {
	search = normalize(search)
	scoreDN := -1
	for _, v := range displays {
		score := adjusted(search, normalize(v))
		// set the smallest non-zero score
		if (score >= 0 && score < scoreDN) || scoreDN == -1 {
			scoreDN = score
		}
	}
	scoreKW := -1
	for _, v := range keywords {
		score := KeywordPenalty * adjusted(search, normalize(v))
		if score == 0 {
			score = KeywordPenalty
		}
		// set the smallest non-zero score
		if (score >= 0 && score < scoreKW) || scoreKW == -1 {
			scoreKW = score
		}
	}

	// if both scores are positive, return the min
	if scoreDN >= 0 && scoreKW >= 0 {
		return min(scoreDN, scoreKW)
	}

	// scoreKW is negative, return scoreDN
	if scoreDN >= 0 {
		return scoreDN
	}
	// scoreDN is negative, return scoreKW
	return scoreKW
}

// removeDiacritics removes "diacritics" :^)
// diacritics are special characters, e.g. GÉANT, becomes GEANT
func removeDiacritics(text string) (string, error) {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, err := transform.String(t, text)
	if err != nil {
		return text, err
	}
	return result, nil
}

// normalize removes diacritics and converts to lower case
func normalize(text string) string {
	dt, _ := removeDiacritics(text)
	return strings.ToLower(dt)
}
