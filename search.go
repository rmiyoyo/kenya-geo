package kenyageo

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

type Kind int

const (
	KindCounty Kind = iota
	KindConstituency
	KindWard
	KindPostOffice
)

func (k Kind) String() string {
	switch k {
	case KindCounty:
		return "county"
	case KindConstituency:
		return "constituency"
	case KindWard:
		return "ward"
	case KindPostOffice:
		return "post office"
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

type searchEntry struct {
	match Match
	norm  string
	runes []rune
}

func (d *Data) buildSearchIndex() {
	add := func(m Match, name string) {
		n := normalize(name)
		d.searchIndex = append(d.searchIndex, searchEntry{match: m, norm: n, runes: []rune(n)})
	}
	for _, c := range d.counties {
		add(Match{Kind: KindCounty, Name: c.Name, Code: strconv.Itoa(c.Code), CountyCode: c.Code}, c.Name)
	}
	for _, c := range d.counties {
		for _, n := range d.constsByCounty[c.Code] {
			add(Match{Kind: KindConstituency, Name: n, CountyCode: c.Code}, n)
		}
	}
	for _, w := range d.wards {
		add(Match{Kind: KindWard, Name: w.Name, Code: w.Code, CountyCode: w.CountyCode}, w.Name)
	}
	for _, p := range d.postOffices {
		add(Match{Kind: KindPostOffice, Name: p.Name, Code: p.Code, CountyCode: p.CountyCode}, p.Name)
	}
}

func (d *Data) Search(query string, limit int) []Match {
	q := normalize(query)
	if q == "" {
		return nil
	}
	qr := []rune(q)
	var scratch []int

	var out []Match
	for i := range d.searchIndex {
		e := &d.searchIndex[i]
		var s float64
		s, scratch = score(q, qr, e.norm, e.runes, scratch)
		if s > 0 {
			m := e.match
			m.Score = s
			out = append(out, m)
		}
	}

	slices.SortStableFunc(out, func(a, b Match) int {
		if c := cmp.Compare(b.Score, a.Score); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Kind, b.Kind); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Name, b.Name); c != 0 {
			return c
		}
		return cmp.Compare(a.CountyCode, b.CountyCode)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func score(q string, qr []rune, n string, nr []rune, scratch []int) (float64, []int) {
	switch {
	case q == n:
		return 1, scratch
	case strings.HasPrefix(n, q):
		return 0.9, scratch
	case strings.Contains(" "+n, " "+q):
		return 0.8, scratch
	case strings.Contains(n, q):
		return 0.7, scratch
	}

	longest := max(len(qr), len(nr))
	if diff := len(qr) - len(nr); float64(max(diff, -diff)) > 0.3*float64(longest) {
		return 0, scratch
	}
	dist, scratch := levenshteinWith(qr, nr, scratch)
	sim := 1 - float64(dist)/float64(longest)
	if sim < 0.7 {
		return 0, scratch
	}
	return 0.6 * sim, scratch
}

func levenshtein(a, b []rune) int {
	d, _ := levenshteinWith(a, b, nil)
	return d
}

func levenshteinWith(a, b []rune, scratch []int) (int, []int) {
	need := 2 * (len(b) + 1)
	if cap(scratch) < need {
		scratch = make([]int, need)
	}
	scratch = scratch[:need]
	prev, cur := scratch[:len(b)+1], scratch[len(b)+1:]
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
	return prev[len(b)], scratch
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
