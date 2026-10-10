package kenyageo

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func FuzzNormalize(f *testing.F) {
	for _, s := range []string{"Nairobi County", "Murang'a", "Chuka/Igambang'ombe", "  Tharaka - Nithi  ", "x county county", "Elgeyo\u200bMarakwet", "Taita–Taveta"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		n := normalize(s)
		if again := normalize(n); again != n {
			t.Errorf("normalize(%q) = %q, but normalize(%q) = %q", s, n, n, again)
		}
		if n != strings.TrimSpace(n) || strings.Contains(n, "  ") {
			t.Errorf("normalize(%q) = %q has stray spaces", s, n)
		}
		if utf8.ValidString(s) && !utf8.ValidString(n) {
			t.Errorf("normalize(%q) = %q is not valid UTF-8", s, n)
		}
	})
}

func FuzzParseAddress(f *testing.F) {
	for _, s := range []string{
		"P.O. Box 123-00100 Nairobi",
		"P. O. BOX 1234 - 20100 NAKURU",
		"Jane Wanjiru\nP.O Box 30100, 00100 GPO Nairobi, Kenya",
		"Private Bag 00100 Nairobi",
		"Box 45, Kisumu",
		"P.O. Box 88, Nakuru 20100",
		"P.O. Box 123-00100 Mombasa",
		"123 Moi Avenue",
	} {
		f.Add(s)
	}
	d := Default()
	f.Fuzz(func(t *testing.T, s string) {
		a, err := d.ParseAddress(s)
		if utf8.ValidString(s) && !utf8.ValidString(a.Town) {
			t.Errorf("ParseAddress(%q): town %q is not valid UTF-8", s, a.Town)
		}
		if err != nil {
			return
		}
		if a.PostOffice == nil || a.PostOffice.Code != a.PostalCode {
			t.Fatalf("ParseAddress(%q) = %+v without a matching post office", s, a)
		}
		b, err := d.ParseAddress(a.String())
		if err != nil {
			t.Fatalf("ParseAddress(%q) printed %q, which does not parse: %v", s, a.String(), err)
		}
		if b.Box != a.Box || b.PostalCode != a.PostalCode || b.PrivateBag != a.PrivateBag || b.Town != a.Town {
			t.Errorf("ParseAddress(%q) = %+v, but its String %q parses as %+v", s, a, a.String(), b)
		}
	})
}

func FuzzSearch(f *testing.F) {
	for _, s := range []string{"nakru", "kibra", "00100", "Murang'a", "x", "county"} {
		f.Add(s, 10)
	}
	d := Default()
	f.Fuzz(func(t *testing.T, q string, limit int) {
		limit %= 50
		ms := d.Search(q, limit)
		if limit > 0 && len(ms) > limit {
			t.Fatalf("Search(%q, %d) returned %d matches", q, limit, len(ms))
		}
		for i, m := range ms {
			if m.Score <= 0 || m.Score > 1 {
				t.Errorf("Search(%q): %s has score %v", q, m.Name, m.Score)
			}
			if i > 0 && m.Score > ms[i-1].Score {
				t.Errorf("Search(%q): results not sorted by score at %d", q, i)
			}
		}
	})
}
