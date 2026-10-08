package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var sources = map[string]string{
	"knbs-county.csv": "https://s3-eu-west-1.amazonaws.com/cfa-openafrica/resources/" +
		"da01bf5a-61d7-43e5-9009-7b54e249e984/kenya-population-land-area-population-density_by_county.csv",
	"knbs-subcounty.csv": "https://s3-eu-west-1.amazonaws.com/cfa-openafrica/resources/" +
		"f1fca3c3-af10-4b55-b33a-95e8fbbd8dc2/kenya-population-by-sub-county.csv",
	"geoboundaries-ADM1.geojson": "https://media.githubusercontent.com/media/wmgeolab/geoBoundaries/" +
		"main/releaseData/gbOpen/KEN/ADM1/geoBoundaries-KEN-ADM1_simplified.geojson",
	"geoboundaries-ADM3.geojson": "https://media.githubusercontent.com/media/wmgeolab/geoBoundaries/" +
		"main/releaseData/gbOpen/KEN/ADM3/geoBoundaries-KEN-ADM3_simplified.geojson",
}

type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type County struct {
	Code               int        `json:"code"`
	Name               string     `json:"name"`
	ISOCode            string     `json:"iso_code"`
	FormerProvince     string     `json:"former_province"`
	Headquarters       string     `json:"headquarters"`
	Population         int        `json:"population"`
	PopulationMale     int        `json:"population_male"`
	PopulationFemale   int        `json:"population_female"`
	PopulationIntersex int        `json:"population_intersex"`
	AreaKm2            float64    `json:"area_km2"`
	Centroid           LatLng     `json:"centroid"`
	BBox               [4]float64 `json:"bbox"`
}

type Ward struct {
	Code                 string  `json:"ward_code"`
	Name                 string  `json:"name"`
	Constituency         string  `json:"constituency_name"`
	CountyCode           int     `json:"county_code"`
	CountyName           string  `json:"county_name"`
	RegisteredVoters2022 *int    `json:"registered_voters_2022"`
	Centroid             *LatLng `json:"centroid"`
}

func main() {
	log.SetFlags(0)
	for name, url := range sources {
		if err := fetch(filepath.Join("data", "raw", name), url); err != nil {
			log.Fatal(err)
		}
	}

	counties, err := buildCounties()
	if err != nil {
		log.Fatal(err)
	}
	wards, err := buildWards(counties)
	if err != nil {
		log.Fatal(err)
	}

	if err := writeLines("data/counties.json", counties); err != nil {
		log.Fatal(err)
	}
	if err := writeLines("data/wards.json", wards); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %d counties and %d wards", len(counties), len(wards))
}

func buildCounties() ([]County, error) {
	var src []struct {
		Code         int    `json:"code"`
		Name         string `json:"name"`
		Headquarters string `json:"headquarters"`
	}
	if err := readJSON("data/source/counties.json", &src); err != nil {
		return nil, err
	}

	census, err := readKNBS()
	if err != nil {
		return nil, err
	}
	shapes, err := readShapes("data/raw/geoboundaries-ADM1.geojson")
	if err != nil {
		return nil, err
	}
	shapeByName := map[string]shape{}
	for _, s := range shapes {
		shapeByName[shapeCountyKey(s.Name)] = s
	}

	out := make([]County, 0, len(src))
	for _, c := range src {
		name := cleanName(c.Name)
		key := norm(name)
		k, ok := census[key]
		if !ok {
			return nil, fmt.Errorf("no KNBS figures for %q", name)
		}
		s, ok := shapeByName[key]
		if !ok {
			return nil, fmt.Errorf("no boundary for %q", name)
		}
		out = append(out, County{
			Code:               c.Code,
			Name:               name,
			ISOCode:            s.ISO,
			FormerProvince:     formerProvince(c.Code),
			Headquarters:       cleanName(c.Headquarters),
			Population:         k.total,
			PopulationMale:     k.male,
			PopulationFemale:   k.female,
			PopulationIntersex: k.intersex,
			AreaKm2:            k.area,
			Centroid:           s.Centroid,
			BBox:               s.BBox,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out, nil
}

func buildWards(counties []County) ([]Ward, error) {
	var src []Ward
	if err := readJSON("data/source/wards.json", &src); err != nil {
		return nil, err
	}
	countyByName := map[string]County{}
	for _, c := range counties {
		countyByName[norm(c.Name)] = c
	}

	countyShapes, err := readShapes("data/raw/geoboundaries-ADM1.geojson")
	if err != nil {
		return nil, err
	}
	wardShapes, err := readShapes("data/raw/geoboundaries-ADM3.geojson")
	if err != nil {
		return nil, err
	}
	shapesInCounty := map[int][]shape{}
	for _, ws := range wardShapes {
		for _, cs := range countyShapes {
			if cs.contains(ws.inside) {
				code := countyByName[shapeCountyKey(cs.Name)].Code
				shapesInCounty[code] = append(shapesInCounty[code], ws)
				break
			}
		}
	}

	matched := 0
	for i := range src {
		w := &src[i]
		applyWardFixes(w)
		w.Name = strings.ToUpper(cleanName(w.Name))
		c, ok := countyByName[norm(w.CountyName)]
		if !ok {
			return nil, fmt.Errorf("ward %s: unknown county %q", w.Code, w.CountyName)
		}
		w.CountyCode, w.CountyName = c.Code, c.Name
		s, ok := matchShape(w.Name, shapesInCounty[c.Code])
		if !ok {
			s, ok = uniqueExact(w.Name, wardShapes, c.BBox, 0.1)
		}
		if ok {
			p := s.Centroid
			w.Centroid = &p
			matched++
		}
	}
	log.Printf("ward centroids: matched %d of %d by name", matched, len(src))
	sort.Slice(src, func(i, j int) bool { return src[i].Code < src[j].Code })
	return src, nil
}

func matchShape(name string, candidates []shape) (shape, bool) {
	n := norm(strings.TrimSuffix(strings.ToLower(name), " ward"))
	var best shape
	bestSim, second := 0.0, 0.0
	for _, s := range candidates {
		sim := similarity(n, norm(s.Name))
		if sim > bestSim {
			bestSim, second, best = sim, bestSim, s
		} else if sim > second {
			second = sim
		}
	}
	if bestSim == 1 || (bestSim >= 0.8 && bestSim-second >= 0.1) {
		return best, true
	}
	return shape{}, false
}

func uniqueExact(name string, shapes []shape, bbox [4]float64, margin float64) (shape, bool) {
	var found []shape
	for _, s := range shapes {
		if norm(s.Name) == norm(name) {
			found = append(found, s)
		}
	}
	if len(found) != 1 {
		return shape{}, false
	}
	c := found[0].Centroid
	inside := c.Lng >= bbox[0]-margin && c.Lng <= bbox[2]+margin &&
		c.Lat >= bbox[1]-margin && c.Lat <= bbox[3]+margin
	return found[0], inside
}

type knbsCounty struct {
	total, male, female, intersex int
	area                          float64
}

func readKNBS() (map[string]knbsCounty, error) {
	rows, err := readCSV("data/raw/knbs-county.csv")
	if err != nil {
		return nil, err
	}
	out := map[string]knbsCounty{}
	byTotal := map[int]string{}
	for _, r := range rows {
		if len(r) < 3 {
			continue
		}
		total, err1 := strconv.Atoi(r[1])
		area, err2 := strconv.ParseFloat(r[2], 64)
		if err1 != nil || err2 != nil || strings.EqualFold(r[0], "Kenya") {
			continue
		}
		key := norm(r[0])
		out[key] = knbsCounty{total: total, area: area}
		byTotal[total] = key
	}

	rows, err = readCSV("data/raw/knbs-subcounty.csv")
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if len(r) < 5 {
			continue
		}
		total, err := strconv.Atoi(strings.TrimSpace(r[4]))
		if err != nil {
			continue
		}
		key, ok := byTotal[total]
		if !ok || out[key].male != 0 {
			continue
		}
		k := out[key]
		k.male, _ = strconv.Atoi(strings.TrimSpace(r[1]))
		k.female, _ = strconv.Atoi(strings.TrimSpace(r[2]))
		k.intersex, _ = strconv.Atoi(strings.TrimSpace(r[3]))
		fixKNBS(key, &k)
		if k.male+k.female+k.intersex != k.total {
			return nil, fmt.Errorf("KNBS %s: %d+%d+%d != %d", key, k.male, k.female, k.intersex, k.total)
		}
		out[key] = k
	}
	for key, k := range out {
		if k.male == 0 {
			return nil, fmt.Errorf("KNBS: no male/female split for %s", key)
		}
	}
	return out, nil
}

func formerProvince(code int) string {
	switch {
	case code <= 6:
		return "Coast"
	case code <= 9:
		return "North Eastern"
	case code <= 17:
		return "Eastern"
	case code <= 22:
		return "Central"
	case code <= 36:
		return "Rift Valley"
	case code <= 40:
		return "Western"
	case code <= 46:
		return "Nyanza"
	}
	return "Nairobi"
}

func fetch(path, url string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	log.Printf("downloading %s", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func readCSV(path string) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	return r.ReadAll()
}

func writeLines[T any](path string, items []T) error {
	var b strings.Builder
	b.WriteString("[\n")
	for i, it := range items {
		line, err := json.Marshal(it)
		if err != nil {
			return err
		}
		b.Write(line)
		if i < len(items)-1 {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	b.WriteString("]\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
