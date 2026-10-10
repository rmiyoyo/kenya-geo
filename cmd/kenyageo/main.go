package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	kenyageo "github.com/rmiyoyo/kenya-geo"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: kenyageo county <name> | postcode <code> | search <query> | at <lat> <lng> | near <lat> <lng> | boundary <ward code or county> | address <postal address>")
		os.Exit(2)
	}
	geo := kenyageo.Default()
	arg := strings.Join(os.Args[2:], " ")

	switch os.Args[1] {
	case "county":
		c, ok := geo.CountyByName(arg)
		if !ok {
			fmt.Fprintf(os.Stderr, "no county called %q\n", arg)
			os.Exit(1)
		}
		fmt.Printf("%d  %s  (HQ %s, former %s Province)\n", c.Code, c.Name, c.Headquarters, c.FormerProvince)
		var names []string
		for _, n := range geo.NeighbouringCounties(c.Code) {
			names = append(names, strings.TrimSuffix(n.Name, " County"))
		}
		fmt.Printf("  borders %s\n", strings.Join(names, ", "))
		v, _ := geo.CountyVoters(c.Code)
		fmt.Printf("  %s registered voters in 2022%s\n", thousands(v.RegisteredVoters2022), incomplete(v))
		for _, name := range geo.Constituencies(c.Code) {
			v, _ := geo.ConstituencyVoters(name)
			fmt.Printf("  %-22s %2d wards %10s voters%s\n", name, v.Wards, thousands(v.RegisteredVoters2022), incomplete(v))
		}
	case "postcode":
		offices := geo.PostOffices(arg)
		if len(offices) == 0 {
			fmt.Fprintf(os.Stderr, "no post office with code %q\n", arg)
			os.Exit(1)
		}
		for _, p := range offices {
			county := "unknown county"
			if c, ok := geo.CountyByCode(p.CountyCode); ok {
				county = c.Name
			}
			fmt.Printf("%s  %s  (%s)\n", p.Code, p.Name, county)
		}
	case "search":
		for _, m := range geo.Search(arg, 10) {
			county, _ := geo.CountyByCode(m.CountyCode)
			fmt.Printf("%.2f  %-12s %-24s %s\n", m.Score, m.Kind, m.Name, county.Name)
		}
	case "at":
		lat, lng, err := parseLatLng(arg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		w, ok := geo.WardAt(lat, lng)
		if !ok {
			c, ok := geo.CountyAt(lat, lng)
			if !ok {
				fmt.Fprintf(os.Stderr, "no ward or county at %v, %v\n", lat, lng)
				os.Exit(1)
			}
			fmt.Printf("%s (no ward boundary here)\n", c.Name)
			return
		}
		fmt.Printf("%s  %s ward, %s constituency, %s\n", w.Code, w.Name, w.Constituency, w.CountyName)
	case "near":
		lat, lng, err := parseLatLng(arg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		for _, p := range geo.PostOfficesNear(lat, lng, 5) {
			county, _ := geo.CountyByCode(p.CountyCode)
			fmt.Printf("%s  %-20s %6.1f km  %s\n", p.Code, p.Name, p.DistanceKm, county.Name)
		}
	case "address":
		a, err := geo.ParseAddress(arg)
		if a.PostOffice != nil {
			county, _ := geo.CountyByCode(a.PostOffice.CountyCode)
			fmt.Printf("%s\n  post office %s %s, %s\n", a, a.PostOffice.Code, a.PostOffice.Name, county.Name)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "boundary":
		var f any
		var ok bool
		if _, err := strconv.Atoi(arg); err == nil && len(arg) == 4 {
			f, ok = geo.WardBoundary(arg)
		} else if c, found := geo.CountyByName(arg); found {
			f, ok = geo.CountyBoundary(c.Code)
		}
		if !ok {
			fmt.Fprintf(os.Stderr, "no boundary for %q\n", arg)
			os.Exit(1)
		}
		json.NewEncoder(os.Stdout).Encode(f)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		os.Exit(2)
	}
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
		return 0, 0, fmt.Errorf("want a latitude and a longitude, got %q", s)
	}
	if lat, err = strconv.ParseFloat(f[0], 64); err != nil {
		return 0, 0, err
	}
	if lng, err = strconv.ParseFloat(f[1], 64); err != nil {
		return 0, 0, err
	}
	return lat, lng, nil
}
