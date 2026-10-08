package main

import (
	"fmt"
	"os"
	"strings"

	kenyageo "github.com/rmiyoyo/kenya-geo"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: kenyageo county <name> | postcode <code> | search <query>")
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
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		os.Exit(2)
	}
}
