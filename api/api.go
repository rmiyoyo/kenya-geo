package api

import (
	_ "embed"
	"encoding/json"
	"errors"
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

type route struct {
	pattern string
	handler func(server, http.ResponseWriter, *http.Request)
}

var routes = []route{
	{"GET /counties", server.counties},
	{"GET /counties/at", server.countyAt},
	{"GET /counties/{county}", server.county},
	{"GET /counties/{county}/boundary", server.countyBoundary},
	{"GET /counties/{county}/constituencies", server.constituencies},
	{"GET /counties/{county}/neighbours", server.neighbours},
	{"GET /counties/{county}/voters", server.countyVoters},
	{"GET /counties/{county}/wards", server.countyWards},
	{"GET /counties/{county}/postoffices", server.countyPostOffices},
	{"GET /constituencies/{name}/voters", server.constituencyVoters},
	{"GET /constituencies/{name}/wards", server.constituencyWards},
	{"GET /voters", server.voters},
	{"GET /wards/{code}", server.ward},
	{"GET /wards/{code}/boundary", server.wardBoundary},
	{"GET /postcodes/{code}", server.postcode},
	{"GET /search", server.search},
	{"GET /addresses", server.address},
	{"GET /at", server.at},
	{"GET /postoffices/near", server.near},
	{"GET /openapi.json", server.openapi},
}

//go:embed openapi.json
var openapiJSON []byte

func New(geo *kenyageo.Data) http.Handler {
	s := server{geo: geo}
	mux := http.NewServeMux()
	for _, rt := range routes {
		mux.HandleFunc(rt.pattern, func(w http.ResponseWriter, r *http.Request) { rt.handler(s, w, r) })
	}
	return mux
}

func (s server) openapi(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(openapiJSON)
}

func (s server) counties(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.geo.Counties())
}

func (s server) county(w http.ResponseWriter, r *http.Request) {
	if c, ok := s.findCounty(w, r); ok {
		writeJSON(w, http.StatusOK, c)
	}
}

func (s server) countyBoundary(w http.ResponseWriter, r *http.Request) {
	c, ok := s.findCounty(w, r)
	if !ok {
		return
	}
	f, ok := s.geo.CountyBoundary(c.Code)
	if !ok {
		writeError(w, http.StatusNotFound, "no boundary for "+c.Name)
		return
	}
	writeGeoJSON(w, f)
}

func (s server) constituencies(w http.ResponseWriter, r *http.Request) {
	if c, ok := s.findCounty(w, r); ok {
		writeJSON(w, http.StatusOK, s.geo.Constituencies(c.Code))
	}
}

func (s server) neighbours(w http.ResponseWriter, r *http.Request) {
	if c, ok := s.findCounty(w, r); ok {
		writeJSON(w, http.StatusOK, s.geo.NeighbouringCounties(c.Code))
	}
}

func (s server) countyVoters(w http.ResponseWriter, r *http.Request) {
	if c, ok := s.findCounty(w, r); ok {
		v, _ := s.geo.CountyVoters(c.Code)
		writeJSON(w, http.StatusOK, v)
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

func (s server) constituencyVoters(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	v, ok := s.geo.ConstituencyVoters(name)
	if !ok {
		writeError(w, http.StatusNotFound, "no constituency "+strconv.Quote(name))
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s server) voters(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.geo.NationalVoters())
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

func (s server) wardBoundary(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	ward, ok := s.geo.WardByCode(code)
	if !ok {
		writeError(w, http.StatusNotFound, "no ward with code "+strconv.Quote(code))
		return
	}
	f, ok := s.geo.WardBoundary(code)
	if !ok {
		writeError(w, http.StatusNotFound, "no boundary for ward "+ward.Code+" "+ward.Name)
		return
	}
	writeGeoJSON(w, f)
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
	limit, ok := parseLimit(w, r, 10)
	if !ok {
		return
	}
	out := []Match{}
	for _, m := range s.geo.Search(q, limit) {
		out = append(out, Match{Kind: m.Kind.String(), Name: m.Name, Code: m.Code, CountyCode: m.CountyCode, Score: m.Score})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s server) address(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	a, err := s.geo.ParseAddress(q)
	switch {
	case errors.Is(err, kenyageo.ErrNoBox):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, kenyageo.ErrUnknownPostOffice):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, kenyageo.ErrTownMismatch):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	case err != nil:
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeJSON(w, http.StatusOK, a)
	}
}

func (s server) at(w http.ResponseWriter, r *http.Request) {
	lat, lng, ok := parseLatLng(w, r)
	if !ok {
		return
	}
	ward, ok := s.geo.WardAt(lat, lng)
	if !ok {
		writeError(w, http.StatusNotFound, "no ward at that point")
		return
	}
	writeJSON(w, http.StatusOK, ward)
}

func (s server) countyAt(w http.ResponseWriter, r *http.Request) {
	lat, lng, ok := parseLatLng(w, r)
	if !ok {
		return
	}
	c, ok := s.geo.CountyAt(lat, lng)
	if !ok {
		writeError(w, http.StatusNotFound, "no county at that point")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s server) near(w http.ResponseWriter, r *http.Request) {
	lat, lng, ok := parseLatLng(w, r)
	if !ok {
		return
	}
	limit, ok := parseLimit(w, r, 5)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.geo.PostOfficesNear(lat, lng, limit))
}

func parseLatLng(w http.ResponseWriter, r *http.Request) (lat, lng float64, ok bool) {
	lat, err1 := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lng, err2 := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	if err1 != nil || err2 != nil {
		writeError(w, http.StatusBadRequest, "lat and lng must be numbers")
		return 0, 0, false
	}
	return lat, lng, true
}

func parseLimit(w http.ResponseWriter, r *http.Request, def int) (int, bool) {
	v := r.URL.Query().Get("limit")
	if v == "" {
		return def, true
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || n > 100 {
		writeError(w, http.StatusBadRequest, "limit must be a number from 1 to 100")
		return 0, false
	}
	return n, true
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	writeTyped(w, status, "application/json", v)
}

func writeGeoJSON(w http.ResponseWriter, v any) {
	writeTyped(w, http.StatusOK, "application/geo+json", v)
}

func writeTyped(w http.ResponseWriter, status int, contentType string, v any) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
