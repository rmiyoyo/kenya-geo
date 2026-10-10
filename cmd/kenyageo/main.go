package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	kenyageo "github.com/rmiyoyo/kenya-geo"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: kenyageo county <name> | postcode <code> | search <query> | at <lat> <lng> | near <lat> <lng>")
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
		for _, name := range geo.Constituencies(c.Code) {
			wards := geo.WardsInConstituency(name)
			fmt.Printf("  %-22s %d wards\n", name, len(wards))
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
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		os.Exit(2)
	}
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
