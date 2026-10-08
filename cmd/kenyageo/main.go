// Command kenyageo is a small CLI over the kenyageo package.
//
//	go run ./cmd/kenyageo county nakuru
//	go run ./cmd/kenyageo search kibra
package main

import (
	"fmt"
	"os"
	"strings"

	kenyageo "github.com/rmiyoyo/kenya-geo"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: kenyageo county <name> | search <query>")
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
		fmt.Printf("%d  %s  (HQ %s, %s region)\n", c.Code, c.Name, c.Headquarters, c.Region)
		for _, name := range geo.Constituencies(c.Code) {
			wards := geo.WardsInConstituency(name)
			fmt.Printf("  %-22s %d wards\n", name, len(wards))
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
