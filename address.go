package kenyageo

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

var (
	ErrNoBox             = errors.New("no P.O. Box or Private Bag found")
	ErrUnknownPostOffice = errors.New("unknown post office")
	ErrTownMismatch      = errors.New("town does not match the postal code")
)

type Address struct {
	Box        string      `json:"box,omitempty"`
	PrivateBag bool        `json:"private_bag"`
	PostalCode string      `json:"postal_code"`
	Town       string      `json:"town,omitempty"`
	PostOffice *PostOffice `json:"post_office"`
}

func (a Address) String() string {
	var b strings.Builder
	if a.PrivateBag {
		b.WriteString("Private Bag")
	} else {
		b.WriteString("P.O. Box")
	}
	if a.Box != "" {
		b.WriteString(" " + a.Box)
		if a.PostalCode != "" {
			b.WriteString("-")
		}
	} else if a.PostalCode != "" {
		b.WriteString(" ")
	}
	b.WriteString(a.PostalCode)
	if a.Town != "" {
		b.WriteString(" " + a.Town)
	}
	return b.String()
}

var boxPrefixes = [][]string{
	{"post", "office", "box"},
	{"p", "o", "box"},
	{"po", "box"},
	{"pobox"},
	{"box"},
}

var bagPrefixes = [][]string{
	{"private", "bag"},
	{"p", "bag"},
	{"pbag"},
}

func (d *Data) ParseAddress(s string) (Address, error) {
	words := strings.Fields(normalize(s))
	var a Address
	start := -1
	for i := range words {
		if n := prefixAt(words[i:], bagPrefixes); n > 0 {
			a.PrivateBag, start = true, i+n
			break
		}
		if n := prefixAt(words[i:], boxPrefixes); n > 0 {
			start = i + n
			break
		}
	}
	if start < 0 {
		return a, ErrNoBox
	}

	rest := words[start:]
	if len(rest) > 0 && isDigits(rest[0]) {
		if len(rest) > 1 && isPostalCode(rest[1]) {
			a.Box, a.PostalCode, rest = rest[0], rest[1], rest[2:]
		} else if a.PrivateBag && isPostalCode(rest[0]) {
			a.PostalCode, rest = rest[0], rest[1:]
		} else {
			a.Box, rest = rest[0], rest[1:]
		}
	}
	rest = slices.DeleteFunc(slices.Clone(rest), func(w string) bool { return w == "gpo" || w == "kenya" })
	if a.PostalCode == "" && len(rest) > 0 && isPostalCode(rest[len(rest)-1]) {
		a.PostalCode, rest = rest[len(rest)-1], rest[:len(rest)-1]
	} else if a.PostalCode == "" && len(rest) > 0 && isPostalCode(rest[0]) {
		a.PostalCode, rest = rest[0], rest[1:]
	}
	town := strings.Join(rest, " ")
	a.Town = titleCase(town)

	if a.PostalCode == "" {
		idx := d.postByName[town]
		if town == "" || len(idx) == 0 {
			return a, fmt.Errorf("%w: no postal code and no post office called %q", ErrUnknownPostOffice, a.Town)
		}
		p := d.postOffices[idx[0]]
		a.PostalCode, a.PostOffice = p.Code, &p
		return a, nil
	}

	idx := d.postByCode[a.PostalCode]
	if len(idx) == 0 {
		return a, fmt.Errorf("%w: %s", ErrUnknownPostOffice, a.PostalCode)
	}
	for _, i := range idx {
		if d.townMatches(town, d.postOffices[i]) {
			p := d.postOffices[i]
			a.PostOffice = &p
			return a, nil
		}
	}
	p := d.postOffices[idx[0]]
	a.PostOffice = &p
	return a, fmt.Errorf("%w: %s is %s, not %s", ErrTownMismatch, p.Code, p.Name, a.Town)
}

func (d *Data) townMatches(town string, p PostOffice) bool {
	key := officeKey(p.Name)
	if town == "" || town == key {
		return true
	}
	if len(key) >= 5 && levenshtein([]rune(town), []rune(key)) <= 1 {
		return true
	}
	c, ok := d.CountyByName(town)
	return ok && c.Code == p.CountyCode
}

func officeKey(name string) string {
	return strings.TrimSuffix(normalize(name), " gpo")
}

func prefixAt(words []string, prefixes [][]string) int {
	for _, p := range prefixes {
		if len(words) >= len(p) && slices.Equal(words[:len(p)], p) {
			return len(p)
		}
	}
	return 0
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

func isPostalCode(s string) bool {
	return len(s) == 5 && isDigits(s)
}

func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}
