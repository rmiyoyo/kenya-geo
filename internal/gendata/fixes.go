package main

import (
	"sort"
	"strings"
	"unicode"
)

func applyWardFixes(w *Ward) {
	switch w.Code {
	case "0555", "0556", "0557", "0558":
		w.Constituency = "Gatundu North"
	case "01811810":
		w.Code = "0181"
	}
}

var postOfficeCountyFixes = map[string]int{
	"00202": 47,
	"50301": 38,
	"50415": 40,
	"50419": 40,
	"30216": 26,
	"50420": 40,
	"90149": 16,
}

func fixKNBS(county string, k *knbsCounty) {
	if county == "nakuru" && k.male == 177272 && k.female == 184835 {
		k.male, k.female = 1077272, 1084835
	}
}

func cleanName(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case r == '–' || r == '—':
			return '-'
		case r == '’' || r == '‘':
			return '\''
		case unicode.Is(unicode.Cf, r):
			return -1
		}
		return r
	}, s)
	return strings.Join(strings.Fields(s), " ")
}

func norm(s string) string {
	var b strings.Builder
	space := false
	for _, r := range strings.ToLower(cleanName(s)) {
		switch {
		case r == '\'' || r == '.':
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			space = false
			b.WriteRune(r)
		default:
			space = true
		}
	}
	return strings.TrimSuffix(b.String(), " county")
}

func shapeCountyKey(name string) string {
	if k := norm(name); k != "tharaka" {
		return k
	}
	return "tharaka nithi"
}

func similarity(a, b string) float64 {
	return max(editSimilarity(a, b), editSimilarity(sortWords(a), sortWords(b)))
}

func sortWords(s string) string {
	w := strings.Fields(s)
	sort.Strings(w)
	return strings.Join(w, " ")
}

func editSimilarity(a, b string) float64 {
	ra, rb := []rune(a), []rune(b)
	longest := max(len(ra), len(rb))
	if longest == 0 {
		return 1
	}
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return 1 - float64(prev[len(rb)])/float64(longest)
}
