package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	kenyageo "github.com/rmiyoyo/kenya-geo"
)

type Match struct {
	Kind       string  `json:"kind"`
	Name       string  `json:"name"`
	Code       string  `json:"code,omitempty"`
	CountyCode int     `json:"county_code"`
	Score      float64 `json:"score"`
}

type server struct {
	geo *kenyageo.Data
}

func New(geo *kenyageo.Data) http.Handler {
	s := server{geo: geo}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /counties", s.counties)
	mux.HandleFunc("GET /counties/{county}", s.county)
	mux.HandleFunc("GET /counties/{county}/constituencies", s.constituencies)
	mux.HandleFunc("GET /counties/{county}/wards", s.countyWards)
	mux.HandleFunc("GET /counties/{county}/postoffices", s.countyPostOffices)
	mux.HandleFunc("GET /constituencies/{name}/wards", s.constituencyWards)
	mux.HandleFunc("GET /wards/{code}", s.ward)
	mux.HandleFunc("GET /postcodes/{code}", s.postcode)
	mux.HandleFunc("GET /search", s.search)
	mux.HandleFunc("GET /at", s.at)
	return mux
}

func (s server) counties(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.geo.Counties())
}

func (s server) county(w http.ResponseWriter, r *http.Request) {
	if c, ok := s.findCounty(w, r); ok {
		writeJSON(w, http.StatusOK, c)
	}
}

func (s server) constituencies(w http.ResponseWriter, r *http.Request) {
	if c, ok := s.findCounty(w, r); ok {
		writeJSON(w, http.StatusOK, s.geo.Constituencies(c.Code))
	}
}

func (s server) countyWards(w http.ResponseWriter, r *http.Request) {
	if c, ok := s.findCounty(w, r); ok {
		writeJSON(w, http.StatusOK, s.geo.WardsInCounty(c.Code))
	}
}

func (s server) countyPostOffices(w http.ResponseWriter, r *http.Request) {
	if c, ok := s.findCounty(w, r); ok {
		writeJSON(w, http.StatusOK, nonNil(s.geo.PostOfficesInCounty(c.Code)))
	}
}

func (s server) findCounty(w http.ResponseWriter, r *http.Request) (kenyageo.County, bool) {
	key := r.PathValue("county")
	var c kenyageo.County
	var ok bool
	if code, err := strconv.Atoi(key); err == nil {
		c, ok = s.geo.CountyByCode(code)
	} else {
		c, ok = s.geo.CountyByName(key)
	}
	if !ok {
		writeError(w, http.StatusNotFound, "no county "+strconv.Quote(key))
	}
	return c, ok
}

func (s server) constituencyWards(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	wards := s.geo.WardsInConstituency(name)
	if len(wards) == 0 {
		writeError(w, http.StatusNotFound, "no constituency "+strconv.Quote(name))
		return
	}
	writeJSON(w, http.StatusOK, wards)
}

func (s server) ward(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	ward, ok := s.geo.WardByCode(code)
	if !ok {
		writeError(w, http.StatusNotFound, "no ward with code "+strconv.Quote(code))
		return
	}
	writeJSON(w, http.StatusOK, ward)
}

func (s server) postcode(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	offices := s.geo.PostOffices(code)
	if len(offices) == 0 {
		writeError(w, http.StatusNotFound, "no post office with code "+strconv.Quote(code))
		return
	}
	writeJSON(w, http.StatusOK, offices)
}

func (s server) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, "missing q")
		return
	}
	limit := 10
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			writeError(w, http.StatusBadRequest, "limit must be a number from 1 to 100")
			return
		}
		limit = n
	}
	out := []Match{}
	for _, m := range s.geo.Search(q, limit) {
		out = append(out, Match{Kind: m.Kind.String(), Name: m.Name, Code: m.Code, CountyCode: m.CountyCode, Score: m.Score})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s server) at(w http.ResponseWriter, r *http.Request) {
	lat, err1 := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lng, err2 := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	if err1 != nil || err2 != nil {
		writeError(w, http.StatusBadRequest, "lat and lng must be numbers")
		return
	}
	ward, ok := s.geo.WardAt(lat, lng)
	if !ok {
		writeError(w, http.StatusNotFound, "no ward at that point")
		return
	}
	writeJSON(w, http.StatusOK, ward)
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
