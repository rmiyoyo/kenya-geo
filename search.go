package kenyageo

import (
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type Kind int

const (
	KindCounty Kind = iota
	KindConstituency
	KindWard
)

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

type Match struct {
	Kind       Kind
	Name       string
	Code       string
	CountyCode int
	Score      float64
}

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

func score(q, n string) float64 {
	switch {
	case q == n:
		return 1
	case strings.HasPrefix(n, q):
		return 0.9
	case strings.Contains(" "+n, " "+q):
		return 0.8
	case strings.Contains(n, q):
		return 0.7
	}

	qr, nr := []rune(q), []rune(n)
	longest := max(len(qr), len(nr))
	sim := 1 - float64(levenshtein(qr, nr))/float64(longest)
	if sim < 0.7 {
		return 0
	}
	return 0.6 * sim
}

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

func normalize(s string) string {
	var b strings.Builder
	pendingSpace := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r == '\'' || r == '’' || r == '‘' || r == '`' || r == '.':
		case unicode.Is(unicode.Cf, r):
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
