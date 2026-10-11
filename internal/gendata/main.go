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
	"geoboundaries-ADM2.geojson": "https://media.githubusercontent.com/media/wmgeolab/geoBoundaries/" +
		"main/releaseData/gbOpen/KEN/ADM2/geoBoundaries-KEN-ADM2_simplified.geojson",
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
	Neighbours         []int      `json:"neighbours"`
	Living             Living     `json:"living"`
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
	for _, set := range []map[string]string{sources, censusSources} {
		for name, url := range set {
			if err := fetch(filepath.Join("data", "raw", name), url); err != nil {
				log.Fatal(err)
			}
		}
	}

	counties, countyShapes, err := buildCounties()
	if err != nil {
		log.Fatal(err)
	}
	wards, shapes, err := buildWards(counties)
	if err != nil {
		log.Fatal(err)
	}
	postOffices, err := buildPostOffices(counties)
	if err != nil {
		log.Fatal(err)
	}
	living, national, err := readLiving(counties)
	if err != nil {
		log.Fatal(err)
	}
	for i := range counties {
		counties[i].Living = living[counties[i].Code]
	}

	if err := writeLines("data/counties.json", counties); err != nil {
		log.Fatal(err)
	}
	if err := writeLines("data/wards.json", wards); err != nil {
		log.Fatal(err)
	}
	if err := writeLines("data/postoffices.json", postOffices); err != nil {
		log.Fatal(err)
	}
	if err := writeLines("data/wardshapes.json", shapes); err != nil {
		log.Fatal(err)
	}
	if err := writeLines("data/countyshapes.json", countyShapes); err != nil {
		log.Fatal(err)
	}
	if err := writeJSON("data/kenya.json", National{Living: national}); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %d counties, %d wards (%d with boundaries) and %d post offices",
		len(counties), len(wards), len(shapes), len(postOffices))
}

func buildCounties() ([]County, []CountyShape, error) {
	var src []struct {
		Code         int    `json:"code"`
		Name         string `json:"name"`
		Headquarters string `json:"headquarters"`
	}
	if err := readJSON("data/source/counties.json", &src); err != nil {
		return nil, nil, err
	}

	census, err := readKNBS()
	if err != nil {
		return nil, nil, err
	}
	shapes, err := readShapes("data/raw/geoboundaries-ADM1.geojson")
	if err != nil {
		return nil, nil, err
	}
	shapeByName := map[string]shape{}
	for _, s := range shapes {
		shapeByName[shapeCountyKey(s.Name)] = s
	}

	out := make([]County, 0, len(src))
	var outShapes []CountyShape
	for _, c := range src {
		name := cleanName(c.Name)
		key := norm(name)
		k, ok := census[key]
		if !ok {
			return nil, nil, fmt.Errorf("no KNBS figures for %q", name)
		}
		s, ok := shapeByName[key]
		if !ok {
			return nil, nil, fmt.Errorf("no boundary for %q", name)
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
		outShapes = append(outShapes, CountyShape{Code: c.Code, Polygons: s.rounded()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	sort.Slice(outShapes, func(i, j int) bool { return outShapes[i].Code < outShapes[j].Code })
	setNeighbours(out, outShapes)
	return out, outShapes, nil
}

type CountyShape struct {
	Code     int       `json:"code"`
	Polygons []polygon `json:"polygons"`
}

func buildWards(counties []County) ([]Ward, []WardShape, error) {
	var src []Ward
	if err := readJSON("data/source/wards.json", &src); err != nil {
		return nil, nil, err
	}
	countyByName := map[string]County{}
	for _, c := range counties {
		countyByName[norm(c.Name)] = c
	}

	countyShapes, err := readShapes("data/raw/geoboundaries-ADM1.geojson")
	if err != nil {
		return nil, nil, err
	}
	constShapes, err := readShapes("data/raw/geoboundaries-ADM2.geojson")
	if err != nil {
		return nil, nil, err
	}
	wardShapes, err := readShapes("data/raw/geoboundaries-ADM3.geojson")
	if err != nil {
		return nil, nil, err
	}
	countyOf := func(pt [2]float64) int {
		for _, cs := range countyShapes {
			if cs.contains(pt) {
				return countyByName[shapeCountyKey(cs.Name)].Code
			}
		}
		return 0
	}
	constsInCounty := map[int][]shape{}
	for _, cs := range constShapes {
		code := countyOf(cs.inside)
		constsInCounty[code] = append(constsInCounty[code], cs)
	}
	shapesInCounty := map[int][]shape{}
	shapesInConst := map[int][]shape{}
	for _, ws := range wardShapes {
		code := countyOf(ws.inside)
		shapesInCounty[code] = append(shapesInCounty[code], ws)
		for _, cs := range constsInCounty[code] {
			if cs.contains(ws.inside) {
				shapesInConst[cs.id] = append(shapesInConst[cs.id], ws)
				break
			}
		}
	}
	constShape := map[string]int{}
	unmatchedConsts := map[string]bool{}

	matched := 0
	var shapes []WardShape
	usedBy := map[int]string{}
	for i := range src {
		w := &src[i]
		applyWardFixes(w)
		w.Name = strings.ToUpper(cleanName(w.Name))
		c, ok := countyByName[norm(w.CountyName)]
		if !ok {
			return nil, nil, fmt.Errorf("ward %s: unknown county %q", w.Code, w.CountyName)
		}
		w.CountyCode, w.CountyName = c.Code, c.Name
		key := fmt.Sprint(c.Code, "/", norm(w.Constituency))
		ci, ok := constShape[key]
		if !ok && !unmatchedConsts[key] {
			if cs, found := matchShape(w.Constituency, constsInCounty[c.Code]); found {
				ci, ok = cs.id, true
				constShape[key] = ci
			} else {
				unmatchedConsts[key] = true
			}
		}
		var s shape
		if ok {
			s, ok = matchShapeIn(w.Name, shapesInConst[ci], shapesInCounty[c.Code])
		}
		if !ok {
			s, ok = matchShape(w.Name, shapesInCounty[c.Code])
		}
		if !ok {
			s, ok = uniqueExact(w.Name, wardShapes, c.BBox, 0.1)
		}
		if ok {
			p := s.Centroid
			w.Centroid = &p
			matched++
			if other, dup := usedBy[s.id]; dup {
				log.Printf("ward boundary %q matched by both %s and %s; boundary kept for %s", s.Name, other, w.Code, other)
				continue
			}
			usedBy[s.id] = w.Code
			shapes = append(shapes, WardShape{Code: w.Code, Polygons: s.rounded()})
		}
	}
	log.Printf("ward centroids: matched %d of %d by name; %d constituencies without a boundary", matched, len(src), len(unmatchedConsts))
	sort.Slice(src, func(i, j int) bool { return src[i].Code < src[j].Code })
	sort.Slice(shapes, func(i, j int) bool { return shapes[i].Code < shapes[j].Code })
	return src, shapes, nil
}

type WardShape struct {
	Code     string    `json:"ward_code"`
	Polygons []polygon `json:"polygons"`
}

func matchShape(name string, candidates []shape) (shape, bool) {
	return matchShapeIn(name, candidates, candidates)
}

func matchShapeIn(name string, preferred, all []shape) (shape, bool) {
	n := norm(strings.TrimSuffix(strings.ToLower(name), " ward"))
	var best shape
	bestSim := 0.0
	for _, s := range preferred {
		if sim := similarity(n, norm(s.Name)); sim > bestSim {
			bestSim, best = sim, s
		}
	}
	second := 0.0
	for _, s := range all {
		if s.id != best.id {
			second = max(second, similarity(n, norm(s.Name)))
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

func writeJSON(path string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
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
