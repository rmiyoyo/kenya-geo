package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const openAfrica = "https://s3-eu-west-1.amazonaws.com/cfa-openafrica/resources/"

var censusSources = map[string]string{
	"knbs-households.csv": openAfrica +
		"111a3d9d-c676-4d5a-9600-d5acb1250b87/population-houseshold-data.csv",
	"knbs-lighting.csv": openAfrica +
		"8ba910a5-c332-4765-9a76-607fe95d3849/percentage-distribution-of-conventional-households-by-main-type-of-lighting-fuel-county-and-sub-.csv",
	"knbs-water.csv": openAfrica +
		"ea9263c3-3c48-4b23-9a79-b175945e690e/percentage-distribution-of-conventional-households-by-main-source-of-drinking-water-county-and-s.csv",
	"knbs-internet.csv": openAfrica +
		"4974e264-8629-4439-87aa-b4b87e975565/distribution-of-population-age-3-years-and-above-using-internet-county-and-sub-county-2019-censu.csv",
	"knbs-mobile.csv": openAfrica +
		"7fe4200a-cccd-41bd-b761-3d424beae754/distribution-of-population-age-3-years-and-above-owning-a-mobile-phone-by-area-of-residence-sex-.csv",
	"knbs-school.csv": openAfrica +
		"9ac966d8-a391-46ab-8cbc-2a14028c668d/distribution-of-population-age-3-years-and-above-by-school-attendance-status-sex-special-age-gro.csv",
	"knbs-urban.csv": openAfrica +
		"a0437750-81a4-4c99-8a4a-d7b95f534a4f/distribution-of-urban-population-by-sex-and-county-2019-census-volume-ii.csv",
	"knbs-rural.csv": openAfrica +
		"06a1d670-fb58-44c6-8d88-f0c18ccc6981/distribution-of-rural-population-by-sex-and-county-2019-census-volume-ii.csv",
	"knbs-religion.csv": openAfrica +
		"2b2a7b48-ff46-4aa1-91c8-94b2c91e568b/distribution-of-population-by-religious-affiliation-and-county-2019-census-volume-iv.csv",
}

type Living struct {
	Households          int     `json:"households"`
	HouseholdSize       float64 `json:"average_household_size"`
	ElectricityPct      float64 `json:"electricity_pct"`
	PipedWaterPct       float64 `json:"piped_water_pct"`
	InternetPct         float64 `json:"internet_pct"`
	MobilePhonePct      float64 `json:"mobile_phone_pct"`
	SchoolAttendancePct float64 `json:"school_attendance_pct"`
}

type Religion struct {
	Total             int `json:"total"`
	Catholic          int `json:"catholic"`
	Protestant        int `json:"protestant"`
	Evangelical       int `json:"evangelical"`
	AfricanInstituted int `json:"african_instituted"`
	Orthodox          int `json:"orthodox"`
	OtherChristian    int `json:"other_christian"`
	Islam             int `json:"islam"`
	Hindu             int `json:"hindu"`
	Traditionist      int `json:"traditionist"`
	OtherReligion     int `json:"other_religion"`
	NoReligion        int `json:"no_religion"`
	DontKnow          int `json:"dont_know"`
	NotStated         int `json:"not_stated"`
}

type National struct {
	PopulationUrban int      `json:"population_urban"`
	PopulationRural int      `json:"population_rural"`
	Living          Living   `json:"living"`
	Religion        Religion `json:"religion"`
}

type urbanRural struct {
	urban, rural int
}

func readUrbanRural(keys []string) (map[string]urbanRural, error) {
	out := map[string]urbanRural{}
	for _, part := range []struct {
		path string
		set  func(*urbanRural, int)
	}{
		{"data/raw/knbs-urban.csv", func(u *urbanRural, n int) { u.urban = n }},
		{"data/raw/knbs-rural.csv", func(u *urbanRural, n int) { u.rural = n }},
	} {
		t, err := readCensusTable(part.path, keys, func(r []string) bool { return strings.TrimSpace(r[0]) == "County" })
		if err != nil {
			return nil, err
		}
		col, err := t.column("Total")
		if err != nil {
			return nil, err
		}
		for k, row := range t.rows {
			n, err := censusNumber(row[col])
			if err != nil {
				return nil, fmt.Errorf("%s %s: %w", part.path, k, err)
			}
			u := out[k]
			part.set(&u, int(n))
			out[k] = u
		}
	}
	return out, nil
}

func readReligion(keys []string) (map[string]Religion, error) {
	const path = "data/raw/knbs-religion.csv"
	t, err := readCensusTable(path, keys, func(r []string) bool { return strings.TrimSpace(r[0]) == "County" })
	if err != nil {
		return nil, err
	}
	columns := []string{"Total", "Catholic", "Protestant", "Evangelical Churches", "African Instituted Churches",
		"Orthodox", "Other Christian", "Islam", "Hindu", "Traditionists", "Other Religion",
		"No religion /Atheists", "Don't Know", "Not Stated"}
	idx := make([]int, len(columns))
	for i, c := range columns {
		if idx[i], err = t.column(c); err != nil {
			return nil, err
		}
	}
	out := map[string]Religion{}
	for k, row := range t.rows {
		v := make([]int, len(idx))
		sum := 0
		for i, c := range idx {
			n, err := censusNumber(row[c])
			if err != nil {
				return nil, fmt.Errorf("%s %s: %w", path, k, err)
			}
			v[i] = int(n)
			if i > 0 {
				sum += v[i]
			}
		}
		if sum != v[0] {
			return nil, fmt.Errorf("%s %s: groups add up to %d, total is %d", path, k, sum, v[0])
		}
		out[k] = Religion{v[0], v[1], v[2], v[3], v[4], v[5], v[6], v[7], v[8], v[9], v[10], v[11], v[12], v[13]}
	}
	return out, nil
}

const kenyaKey = "kenya"

func censusKey(name string) string {
	switch k := norm(name); k {
	case "nairobi city":
		return "nairobi"
	case "elg eyo marakwet":
		return "elgeyo marakwet"
	default:
		return k
	}
}

type censusTable struct {
	header []string
	rows   map[string][]string
}

func readCensusTable(path string, keys []string, header func([]string) bool) (censusTable, error) {
	rows, err := readCSV(path)
	if err != nil {
		return censusTable{}, err
	}
	want := map[string]bool{kenyaKey: true}
	for _, k := range keys {
		want[k] = true
	}
	t := censusTable{rows: map[string][]string{}}
	for _, r := range rows {
		if len(r) == 0 {
			continue
		}
		if t.header == nil && header(r) {
			t.header = r
			continue
		}
		k := censusKey(r[0])
		if want[k] {
			if _, seen := t.rows[k]; !seen {
				t.rows[k] = r
			}
		}
	}
	if t.header == nil {
		return censusTable{}, fmt.Errorf("%s: no header row", path)
	}
	for _, k := range keys {
		if _, ok := t.rows[k]; !ok {
			return censusTable{}, fmt.Errorf("%s: no row for %q", path, k)
		}
	}
	return t, nil
}

func (t censusTable) column(name string) (int, error) {
	for i, h := range t.header {
		if strings.EqualFold(strings.TrimSpace(h), name) {
			return i, nil
		}
	}
	return 0, fmt.Errorf("no column %q in %q", name, t.header)
}

func censusNumber(s string) (float64, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	if s == "-" {
		return 0, nil
	}
	return strconv.ParseFloat(s, 64)
}

func round1(f float64) float64 { return math.Round(f*10) / 10 }

func readLiving(counties []County) (map[int]Living, Living, error) {
	keys := make([]string, len(counties))
	for i, c := range counties {
		keys[i] = censusKey(c.Name)
	}
	out := map[string]*Living{kenyaKey: {}}
	for _, k := range keys {
		out[k] = &Living{}
	}

	startsWith := func(prefix string) func([]string) bool {
		return func(r []string) bool { return strings.HasPrefix(strings.TrimSpace(r[0]), prefix) }
	}

	households, err := readCensusTable("data/raw/knbs-households.csv", keys, startsWith("name"))
	if err != nil {
		return nil, Living{}, err
	}
	popCol, err := households.column("Population")
	if err != nil {
		return nil, Living{}, err
	}
	hhCol, err := households.column("No.of Households")
	if err != nil {
		return nil, Living{}, err
	}
	for k, l := range out {
		pop, err1 := censusNumber(households.rows[k][popCol])
		hh, err2 := censusNumber(households.rows[k][hhCol])
		if err1 != nil || err2 != nil || hh == 0 {
			return nil, Living{}, fmt.Errorf("households %s: %v %v", k, err1, err2)
		}
		l.Households = int(hh)
		l.HouseholdSize = round1(pop / hh)
	}

	percent := func(path string, header func([]string) bool, columns []string, set func(*Living, float64)) error {
		t, err := readCensusTable(path, keys, header)
		if err != nil {
			return err
		}
		var idx []int
		for _, c := range columns {
			i, err := t.column(c)
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			idx = append(idx, i)
		}
		for k, l := range out {
			sum := 0.0
			for _, i := range idx {
				v, err := censusNumber(t.rows[k][i])
				if err != nil {
					return fmt.Errorf("%s %s: %w", path, k, err)
				}
				sum += v
			}
			if sum < 0 || sum > 100 {
				return fmt.Errorf("%s %s: %v is not a percentage", path, k, sum)
			}
			set(l, round1(sum))
		}
		return nil
	}

	if err := percent("data/raw/knbs-lighting.csv", startsWith("County/"), []string{"Mains Electricity"},
		func(l *Living, v float64) { l.ElectricityPct = v }); err != nil {
		return nil, Living{}, err
	}
	if err := percent("data/raw/knbs-water.csv", startsWith("County/"), []string{"Piped into dwelling", "Piped to yard/ Plot"},
		func(l *Living, v float64) { l.PipedWaterPct = v }); err != nil {
		return nil, Living{}, err
	}

	perCent := func(path string, set func(*Living, float64)) error {
		t, err := readCensusTable(path, keys, startsWith("Sub-County"))
		if err != nil {
			return err
		}
		col := -1
		for i, h := range t.header {
			if strings.EqualFold(strings.TrimSpace(h), "per cent") {
				col = i
				break
			}
		}
		if col < 0 {
			return fmt.Errorf("%s: no per cent column", path)
		}
		for k, l := range out {
			v, err := censusNumber(t.rows[k][col])
			if err != nil || v < 0 || v > 100 {
				return fmt.Errorf("%s %s: bad percentage %q", path, k, t.rows[k][col])
			}
			set(l, v)
		}
		return nil
	}
	if err := perCent("data/raw/knbs-internet.csv", func(l *Living, v float64) { l.InternetPct = v }); err != nil {
		return nil, Living{}, err
	}
	if err := perCent("data/raw/knbs-mobile.csv", func(l *Living, v float64) { l.MobilePhonePct = v }); err != nil {
		return nil, Living{}, err
	}

	attending, err := readSchoolAttendance("data/raw/knbs-school.csv", keys)
	if err != nil {
		return nil, Living{}, err
	}
	var natAt, natTotal float64
	for _, k := range keys {
		a := attending[k]
		out[k].SchoolAttendancePct = round1(100 * a[0] / a[1])
		natAt += a[0]
		natTotal += a[1]
	}
	out[kenyaKey].SchoolAttendancePct = round1(100 * natAt / natTotal)

	byCode := map[int]Living{}
	for i, c := range counties {
		byCode[c.Code] = *out[keys[i]]
	}
	return byCode, *out[kenyaKey], nil
}

func readSchoolAttendance(path string, keys []string) (map[string][2]float64, error) {
	rows, err := readCSV(path)
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, k := range keys {
		want[k] = true
	}
	out := map[string][2]float64{}
	current := ""
	for _, r := range rows {
		if len(r) < 5 {
			continue
		}
		label := strings.TrimSpace(r[0])
		if k := censusKey(label); want[k] {
			if _, seen := out[k]; !seen {
				current = k
				out[k] = [2]float64{}
			} else {
				current = ""
			}
			continue
		}
		if current == "" || (label != "6-13" && label != "14-17") {
			continue
		}
		total, err1 := censusNumber(r[1])
		at, err2 := censusNumber(r[4])
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("%s %s %s: %v %v", path, current, label, err1, err2)
		}
		v := out[current]
		out[current] = [2]float64{v[0] + at, v[1] + total}
	}
	for _, k := range keys {
		if out[k][1] == 0 {
			return nil, fmt.Errorf("%s: no school-age rows for %q", path, k)
		}
	}
	return out, nil
}
