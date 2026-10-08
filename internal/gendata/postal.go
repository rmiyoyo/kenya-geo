package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
)

type PostOffice struct {
	Code       string  `json:"postal_code"`
	Name       string  `json:"name"`
	CountyCode int     `json:"county_code"`
	Location   *LatLng `json:"location"`
}

func buildPostOffices(counties []County) ([]PostOffice, error) {
	f, err := os.Open("data/source/geonames-KE.txt")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = '\t'
	r.LazyQuotes = true
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	countyByKey := map[string]int{}
	for _, c := range counties {
		countyByKey[norm(c.Name)] = c.Code
	}
	shapes, err := readShapes("data/raw/geoboundaries-ADM1.geojson")
	if err != nil {
		return nil, err
	}
	var nairobi shape
	for _, s := range shapes {
		if norm(s.Name) == "nairobi" {
			nairobi = s
		}
	}

	out := make([]PostOffice, 0, len(rows))
	unknown, moved := 0, 0
	for _, row := range rows {
		if len(row) < 12 {
			return nil, fmt.Errorf("geonames: short row %q", row)
		}
		lat, err1 := strconv.ParseFloat(row[9], 64)
		lng, err2 := strconv.ParseFloat(row[10], 64)
		accuracy, err3 := strconv.Atoi(row[11])
		if err1 != nil || err2 != nil || err3 != nil {
			return nil, fmt.Errorf("geonames: bad coordinates in %q", row)
		}
		code, ok := countyByKey[geonamesCountyKey(row[3])]
		if !ok {
			return nil, fmt.Errorf("geonames: unknown county %q", row[3])
		}
		p := PostOffice{Code: row[1], Name: cleanName(row[2]), CountyCode: code}
		precise := accuracy >= 3
		if precise {
			p.Location = &LatLng{Lat: lat, Lng: lng}
		}
		pt := [2]float64{lng, lat}
		if code == 47 && !nairobiHeadOffice(p.Code) && !nairobi.contains(pt) {
			p.CountyCode = 0
			if precise {
				p.CountyCode = countyAt(pt, shapes, countyByKey)
			}
			if p.CountyCode == 0 {
				unknown++
			} else {
				moved++
			}
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		return out[i].Name < out[j].Name
	})
	log.Printf("post offices: %d, %d moved out of Nairobi, %d with unknown county", len(out), moved, unknown)
	return out, nil
}

func nairobiHeadOffice(code string) bool {
	return len(code) == 5 && code[:2] == "00" && code[2] >= '1' && code[2] <= '8' && code[3:] == "00"
}

func geonamesCountyKey(name string) string {
	switch k := norm(name); k {
	case "elegeyo marakwet":
		return "elgeyo marakwet"
	default:
		return k
	}
}

func countyAt(pt [2]float64, shapes []shape, countyByKey map[string]int) int {
	for _, s := range shapes {
		if s.contains(pt) {
			return countyByKey[shapeCountyKey(s.Name)]
		}
	}
	return 0
}
