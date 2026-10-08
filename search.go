package kenyageo

import (
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// Kind says what a search Match refers to.
type Kind int

// iota counts up from 0 inside a const block; it is Go's usual way to
// declare an enum-like set of values.
const (
	KindCounty Kind = iota
	KindConstituency
	KindWard
)

// String makes Kind satisfy the fmt.Stringer interface, so fmt.Println
// prints "ward" instead of "2". Go interfaces are satisfied implicitly:
// there is no "implements" keyword.
func (k Kind) String() string {
	switch k {
	case KindCounty:
		return "county"
	case KindConstituency:
		return "constituency"
	case KindWard:
		return "ward"
	}
	return "unknown"
}

// Match is one search result.
type Match struct {
	Kind       Kind
	Name       string  // display name as in the source data
	Code       string  // ward code for wards, county code for counties, empty for constituencies
	CountyCode int     // the county the match belongs to
	Score      float64 // 1 is an exact match; higher is better
}

// Search finds counties, constituencies and wards whose names resemble the
// query, best match first, returning at most limit results (limit <= 0
// means no limit). It tolerates case, punctuation and small typos:
// "nyeri", "kibera", "Lari Kirenga" and "makadara" all work.
func (d *Data) Search(query string, limit int) []Match {
	q := normalize(query)
	if q == "" {
		return nil
	}

	var out []Match
	consider := func(m Match, name string) {
		if s := score(q, normalize(name)); s > 0 {
			m.Score = s
			out = append(out, m)
		}
	}

	for _, c := range d.counties {
		consider(Match{Kind: KindCounty, Name: c.Name, Code: strconv.Itoa(c.Code), CountyCode: c.Code}, c.Name)
	}
	for code, names := range d.constsByCounty {
		for _, n := range names {
			consider(Match{Kind: KindConstituency, Name: n, CountyCode: code}, n)
		}
	}
	for _, w := range d.wards {
		consider(Match{Kind: KindWard, Name: w.Name, Code: w.Code, CountyCode: w.CountyCode}, w.Name)
	}

	// sort.SliceStable keeps the deterministic tie-break order below even
	// though map iteration order (constsByCounty) is random in Go.
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.CountyCode < b.CountyCode
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// score rates how well the normalized query q matches the normalized name
// n, from 0 (no match) to 1 (identical).
func score(q, n string) float64 {
	switch {
	case q == n:
		return 1
	case strings.HasPrefix(n, q):
		return 0.9
	case strings.Contains(" "+n, " "+q): // query starts a later word
		return 0.8
	case strings.Contains(n, q):
		return 0.7
	}
	// Fall back to edit distance for typos ("nakru" -> "nakuru"). Only
	// close matches count, scaled below every substring match.
	qr, nr := []rune(q), []rune(n)
	longest := max(len(qr), len(nr))
	sim := 1 - float64(levenshtein(qr, nr))/float64(longest)
	if sim < 0.7 {
		return 0
	}
	return 0.6 * sim
}

// levenshtein counts the single-character edits (insert, delete, replace)
// needed to turn a into b. It works on runes, not bytes, so a non-ASCII
// letter counts as one character. Only two rows of the classic table are
// kept, which is all the algorithm needs.
func levenshtein(a, b []rune) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

// normalize turns a place name into a matching key: lower case, apostrophes
// (straight or curly), dots and invisible characters dropped, every run of
// other separators (spaces, hyphens, en dashes, slashes) collapsed to one
// space, and a trailing " county" removed.
//
//	"Murang’a County" -> "muranga"
//	"Taita–Taveta"    -> "taita taveta"
//	"LARI/KIRENGA"    -> "lari kirenga"
func normalize(s string) string {
	var b strings.Builder
	pendingSpace := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r == '\'' || r == '’' || r == '‘' || r == '`' || r == '.':
			// dropped: "Murang'a" and "Muranga" should be equal
		case unicode.Is(unicode.Cf, r):
			// format characters such as the zero-width space hiding in "TAMB​UA"
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if pendingSpace && b.Len() > 0 {
				b.WriteByte(' ')
			}
			pendingSpace = false
			b.WriteRune(r)
		default:
			pendingSpace = true
		}
	}
	return strings.TrimSuffix(b.String(), " county")
}
