package main

import (
	"log"
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

type osmPostOffice struct {
	Ref    string
	Name   string
	tokens []string
	codes  map[string]bool
	pt     [2]float64
}

var (
	osmBracketed  = regexp.MustCompile(`\([^)]*\)`)
	osmPostalCode = regexp.MustCompile(`\b\d{5}\b`)
	osmStopWords  = map[string]bool{
		"post": true, "office": true, "postal": true, "posta": true, "postoffice": true,
		"gpo": true, "g": true, "p": true, "o": true, "po": true, "general": true,
		"branch": true, "kenya": true, "corporation": true, "of": true,
	}
)

func readOSMPostOffices(path string) ([]osmPostOffice, error) {
	var src struct {
		Elements []struct {
			Type   string                      `json:"type"`
			ID     int64                       `json:"id"`
			Lat    float64                     `json:"lat"`
			Lon    float64                     `json:"lon"`
			Center *struct{ Lat, Lon float64 } `json:"center"`
			Tags   map[string]string           `json:"tags"`
		} `json:"elements"`
	}
	if err := readJSON(path, &src); err != nil {
		return nil, err
	}
	var out []osmPostOffice
	for _, e := range src.Elements {
		name := e.Tags["name"]
		tokens := postOfficeTokens(name)
		if len(tokens) == 0 {
			continue
		}
		lat, lng := e.Lat, e.Lon
		if e.Center != nil {
			lat, lng = e.Center.Lat, e.Center.Lon
		}
		codes := map[string]bool{}
		for _, k := range []string{"name", "addr:postcode", "postal_code"} {
			for _, c := range osmPostalCode.FindAllString(e.Tags[k], -1) {
				codes[c] = true
			}
		}
		out = append(out, osmPostOffice{
			Ref:    e.Type + "/" + strconv.FormatInt(e.ID, 10),
			Name:   name,
			tokens: tokens,
			codes:  codes,
			pt:     [2]float64{lng, lat},
		})
	}
	return out, nil
}

func postOfficeTokens(name string) []string {
	var out []string
	for _, w := range strings.Fields(norm(osmBracketed.ReplaceAllString(name, " "))) {
		if !osmStopWords[w] && !osmPostalCode.MatchString(w) {
			out = append(out, w)
		}
	}
	return out
}

const (
	osmNoMatch = iota
	osmSubsetMatch
	osmNameMatch
	osmCodeMatch
)

func osmMatch(p PostOffice, o osmPostOffice) int {
	if len(o.codes) > 0 && !o.codes[p.Code] {
		return osmNoMatch
	}
	tokens := postOfficeTokens(p.Name)
	if len(tokens) == 0 {
		return osmNoMatch
	}
	switch {
	case strings.Join(tokens, "") == strings.Join(o.tokens, ""):
		if o.codes[p.Code] {
			return osmCodeMatch
		}
		return osmNameMatch
	case o.codes[p.Code] && (isPrefix(o.tokens, tokens) || isPrefix(tokens, o.tokens)):
		return osmCodeMatch
	case len(o.codes) == 0 && isSubset(o.tokens, tokens):
		return osmSubsetMatch
	}
	return osmNoMatch
}

func isPrefix(a, b []string) bool {
	return len(a) <= len(b) && slices.Equal(a, b[:len(a)])
}

func isSubset(a, b []string) bool {
	for _, w := range a {
		if !slices.Contains(b, w) {
			return false
		}
	}
	return true
}

func addOSMLocations(offices []PostOffice, osm []osmPostOffice, shapes []shape, countyByKey map[string]int) {
	score := make([][]int, len(offices))
	bestForOffice := make([]int, len(offices))
	bestForOSM := make([]int, len(osm))
	for i, p := range offices {
		score[i] = make([]int, len(osm))
		for j, o := range osm {
			s := osmMatch(p, o)
			score[i][j] = s
			bestForOffice[i] = max(bestForOffice[i], s)
			bestForOSM[j] = max(bestForOSM[j], s)
		}
	}

	var added, ambiguous, wrongCounty, crowded []string
	for i := range offices {
		p := &offices[i]
		if p.Location != nil || bestForOffice[i] == osmNoMatch {
			continue
		}
		var picks []int
		for j := range osm {
			if score[i][j] == bestForOffice[i] && bestForOSM[j] == score[i][j] {
				picks = append(picks, j)
			}
		}
		if len(picks) == 0 {
			continue
		}
		if len(picks) > 1 || rivals(score, picks[0], i) {
			ambiguous = append(ambiguous, p.Code+" "+p.Name)
			continue
		}
		o := osm[picks[0]]
		if score[i][picks[0]] != osmCodeMatch {
			if q := nearestGeoNamesOffice(offices, o.pt); q != nil {
				crowded = append(crowded, p.Code+" "+p.Name+" ~ "+o.Name+" is beside "+q.Code+" "+q.Name)
				continue
			}
		}
		if c := countyAt(o.pt, shapes, countyByKey); c != 0 && c != p.CountyCode {
			wrongCounty = append(wrongCounty, p.Code+" "+p.Name+" ~ "+o.Name)
			continue
		}
		p.Location = &LatLng{Lat: round4(o.pt[1]), Lng: round4(o.pt[0])}
		p.LocationSource = "openstreetmap"
		added = append(added, p.Code+" "+p.Name+" ~ "+o.Name+" ("+o.Ref+")")
	}
	sort.Strings(added)
	log.Printf("post offices: %d locations added from OpenStreetMap, %d ambiguous, %d in another county, %d beside another office",
		len(added), len(ambiguous), len(wrongCounty), len(crowded))
	for _, s := range added {
		log.Printf("  osm: %s", s)
	}
	for _, s := range ambiguous {
		log.Printf("  osm ambiguous: %s", s)
	}
	for _, s := range wrongCounty {
		log.Printf("  osm wrong county: %s", s)
	}
	for _, s := range crowded {
		log.Printf("  osm beside another office: %s", s)
	}
}

func rivals(score [][]int, j, office int) bool {
	for i := range score {
		if i != office && score[i][j] == score[office][j] {
			return true
		}
	}
	return false
}

func nearestGeoNamesOffice(offices []PostOffice, pt [2]float64) *PostOffice {
	for i, q := range offices {
		if q.LocationSource == "geonames" && distanceKm(pt[1], pt[0], q.Location.Lat, q.Location.Lng) < 0.3 {
			return &offices[i]
		}
	}
	return nil
}

func distanceKm(lat1, lng1, lat2, lng2 float64) float64 {
	const r = 6371.0
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * r * math.Asin(math.Sqrt(a))
}

func round4(f float64) float64 {
	return math.Round(f*1e4) / 1e4
}
