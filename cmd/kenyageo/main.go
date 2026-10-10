package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	kenyageo "github.com/rmiyoyo/kenya-geo"
)

const usage = `usage: kenyageo [-json] <command> <argument>

commands:
  county <name>                  constituencies, voters and neighbours
  postcode <code>                post offices with that postal code
  search <query>                 fuzzy search
  at <lat> <lng>                 ward, constituency and county for a point
  near <lat> <lng>               the five closest post offices
  boundary <ward code|county>    outline as GeoJSON
  address <postal address>       parse and check a postal address
`

type result struct {
	value any
	text  func(w io.Writer)
}

type usageError string

func (e usageError) Error() string { return string(e) }

type command func(geo *kenyageo.Data, arg string) (result, error)

var commands = map[string]command{
	"county":   county,
	"postcode": postcode,
	"search":   search,
	"at":       at,
	"near":     near,
	"boundary": boundary,
	"address":  address,
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("kenyageo", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usage) }
	asJSON := fs.Bool("json", false, "print JSON instead of text")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() < 2 {
		fs.Usage()
		return 2
	}
	cmd, ok := commands[fs.Arg(0)]
	if !ok {
		fmt.Fprintf(stderr, "unknown command %q\n\n%s", fs.Arg(0), usage)
		return 2
	}

	res, err := cmd(kenyageo.Default(), strings.Join(fs.Args()[1:], " "))
	if res.value != nil {
		if *asJSON || res.text == nil {
			enc := json.NewEncoder(stdout)
			enc.SetIndent("", "  ")
			enc.Encode(res.value)
		} else {
			res.text(stdout)
		}
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		var u usageError
		if errors.As(err, &u) {
			return 2
		}
		return 1
	}
	return 0
}

type countyReport struct {
	kenyageo.County
	Voters         kenyageo.VoterTotal  `json:"voters"`
	Constituencies []constituencyReport `json:"constituencies"`
}

type constituencyReport struct {
	Name   string              `json:"name"`
	Voters kenyageo.VoterTotal `json:"voters"`
}

func county(geo *kenyageo.Data, arg string) (result, error) {
	c, ok := geo.CountyByName(arg)
	if !ok {
		return result{}, fmt.Errorf("no county called %q", arg)
	}
	r := countyReport{County: c}
	r.Voters, _ = geo.CountyVoters(c.Code)
	for _, name := range geo.Constituencies(c.Code) {
		v, _ := geo.ConstituencyVoters(name)
		r.Constituencies = append(r.Constituencies, constituencyReport{name, v})
	}
	return result{r, func(w io.Writer) {
		fmt.Fprintf(w, "%d  %s  (HQ %s, former %s Province)\n", c.Code, c.Name, c.Headquarters, c.FormerProvince)
		var names []string
		for _, n := range geo.NeighbouringCounties(c.Code) {
			names = append(names, strings.TrimSuffix(n.Name, " County"))
		}
		fmt.Fprintf(w, "  borders %s\n", strings.Join(names, ", "))
		fmt.Fprintf(w, "  %s registered voters in 2022%s\n", thousands(r.Voters.RegisteredVoters2022), incomplete(r.Voters))
		for _, cr := range r.Constituencies {
			v := cr.Voters
			fmt.Fprintf(w, "  %-22s %2d wards %10s voters%s\n", cr.Name, v.Wards, thousands(v.RegisteredVoters2022), incomplete(v))
		}
	}}, nil
}

func postcode(geo *kenyageo.Data, arg string) (result, error) {
	offices := geo.PostOffices(arg)
	if len(offices) == 0 {
		return result{}, fmt.Errorf("no post office with code %q", arg)
	}
	return result{offices, func(w io.Writer) {
		for _, p := range offices {
			county := "unknown county"
			if c, ok := geo.CountyByCode(p.CountyCode); ok {
				county = c.Name
			}
			fmt.Fprintf(w, "%s  %s  (%s)\n", p.Code, p.Name, county)
		}
	}}, nil
}

type searchResult struct {
	Kind       string  `json:"kind"`
	Name       string  `json:"name"`
	Code       string  `json:"code,omitempty"`
	CountyCode int     `json:"county_code"`
	Score      float64 `json:"score"`
}

func search(geo *kenyageo.Data, arg string) (result, error) {
	out := []searchResult{}
	for _, m := range geo.Search(arg, 10) {
		out = append(out, searchResult{m.Kind.String(), m.Name, m.Code, m.CountyCode, m.Score})
	}
	return result{out, func(w io.Writer) {
		for _, m := range out {
			county, _ := geo.CountyByCode(m.CountyCode)
			fmt.Fprintf(w, "%.2f  %-12s %-24s %s\n", m.Score, m.Kind, m.Name, county.Name)
		}
	}}, nil
}

type location struct {
	Ward   *kenyageo.Ward  `json:"ward"`
	County kenyageo.County `json:"county"`
}

func at(geo *kenyageo.Data, arg string) (result, error) {
	lat, lng, err := parseLatLng(arg)
	if err != nil {
		return result{}, err
	}
	if w, ok := geo.WardAt(lat, lng); ok {
		c, _ := geo.CountyByCode(w.CountyCode)
		return result{location{&w, c}, func(out io.Writer) {
			fmt.Fprintf(out, "%s  %s ward, %s constituency, %s\n", w.Code, w.Name, w.Constituency, w.CountyName)
		}}, nil
	}
	c, ok := geo.CountyAt(lat, lng)
	if !ok {
		return result{}, fmt.Errorf("no ward or county at %v, %v", lat, lng)
	}
	return result{location{nil, c}, func(out io.Writer) {
		fmt.Fprintf(out, "%s (no ward boundary here)\n", c.Name)
	}}, nil
}

func near(geo *kenyageo.Data, arg string) (result, error) {
	lat, lng, err := parseLatLng(arg)
	if err != nil {
		return result{}, err
	}
	offices := geo.PostOfficesNear(lat, lng, 5)
	return result{offices, func(w io.Writer) {
		for _, p := range offices {
			county, _ := geo.CountyByCode(p.CountyCode)
			fmt.Fprintf(w, "%s  %-20s %6.1f km  %s\n", p.Code, p.Name, p.DistanceKm, county.Name)
		}
	}}, nil
}

func address(geo *kenyageo.Data, arg string) (result, error) {
	a, err := geo.ParseAddress(arg)
	if a.PostOffice == nil {
		return result{}, err
	}
	return result{a, func(w io.Writer) {
		county, _ := geo.CountyByCode(a.PostOffice.CountyCode)
		fmt.Fprintf(w, "%s\n  post office %s %s, %s\n", a, a.PostOffice.Code, a.PostOffice.Name, county.Name)
	}}, err
}

func boundary(geo *kenyageo.Data, arg string) (result, error) {
	var f any
	var ok bool
	if _, err := strconv.Atoi(arg); err == nil && len(arg) == 4 {
		f, ok = geo.WardBoundary(arg)
	} else if c, found := geo.CountyByName(arg); found {
		f, ok = geo.CountyBoundary(c.Code)
	}
	if !ok {
		return result{}, fmt.Errorf("no boundary for %q", arg)
	}
	return result{value: f}, nil
}

func thousands(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func incomplete(v kenyageo.VoterTotal) string {
	if v.Complete {
		return ""
	}
	return " (a ward has no figure)"
}

func parseLatLng(s string) (lat, lng float64, err error) {
	f := strings.Fields(strings.ReplaceAll(s, ",", " "))
	if len(f) != 2 {
		return 0, 0, usageError(fmt.Sprintf("want a latitude and a longitude, got %q", s))
	}
	if lat, err = strconv.ParseFloat(f[0], 64); err != nil {
		return 0, 0, usageError(err.Error())
	}
	if lng, err = strconv.ParseFloat(f[1], 64); err != nil {
		return 0, 0, usageError(err.Error())
	}
	return lat, lng, nil
}
