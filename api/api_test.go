package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	kenyageo "github.com/rmiyoyo/kenya-geo"
)

var handler = New(kenyageo.Default())

func get(t *testing.T, path string, v any) int {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if v != nil && rec.Code == http.StatusOK {
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("%s: Content-Type = %q", path, ct)
		}
		if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
			t.Fatalf("%s: %v\n%s", path, err, rec.Body)
		}
	}
	return rec.Code
}

func TestCounties(t *testing.T) {
	var counties []kenyageo.County
	if code := get(t, "/counties", &counties); code != http.StatusOK || len(counties) != 47 {
		t.Fatalf("status %d, %d counties", code, len(counties))
	}
	for _, path := range []string{"/counties/32", "/counties/nakuru", "/counties/Nakuru%20County"} {
		var c kenyageo.County
		if code := get(t, path, &c); code != http.StatusOK || c.Code != 32 {
			t.Errorf("%s: status %d, county %d", path, code, c.Code)
		}
	}
}

func TestCountyLists(t *testing.T) {
	var consts []string
	if get(t, "/counties/22/constituencies", &consts); len(consts) != 12 {
		t.Errorf("Kiambu has %d constituencies, want 12", len(consts))
	}
	var neighbours []kenyageo.County
	if get(t, "/counties/nairobi/neighbours", &neighbours); len(neighbours) != 3 {
		t.Errorf("Nairobi has %d neighbours, want 3", len(neighbours))
	}
	var wards []kenyageo.Ward
	if get(t, "/counties/mombasa/wards", &wards); len(wards) != 30 {
		t.Errorf("Mombasa has %d wards, want 30", len(wards))
	}
	var offices []kenyageo.PostOffice
	if get(t, "/counties/47/postoffices", &offices); len(offices) == 0 {
		t.Error("Nairobi has no post offices")
	}
}

func TestConstituencyWards(t *testing.T) {
	for path, want := range map[string]int{
		"/constituencies/Kibra/wards":                 5,
		"/constituencies/gatundu%20north/wards":       4,
		"/constituencies/Chuka%2FIgambang'ombe/wards": 5,
		"/constituencies/chuka-igambangombe/wards":    5,
	} {
		var wards []kenyageo.Ward
		if code := get(t, path, &wards); code != http.StatusOK || len(wards) != want {
			t.Errorf("%s: status %d, %d wards, want %d", path, code, len(wards), want)
		}
	}
}

func TestLookups(t *testing.T) {
	var w kenyageo.Ward
	if get(t, "/wards/0181", &w); w.Name != "DELLA" {
		t.Errorf("ward 0181 = %q, want DELLA", w.Name)
	}
	var offices []kenyageo.PostOffice
	if get(t, "/postcodes/00100", &offices); len(offices) != 1 || offices[0].CountyCode != 47 {
		t.Errorf("00100 = %+v", offices)
	}
	if get(t, "/at?lat=-1.2884&lng=36.8233", &w); w.Code != "1439" {
		t.Errorf("at KICC = %s %s, want 1439", w.Code, w.Name)
	}
}

func TestNear(t *testing.T) {
	var offices []kenyageo.NearbyPostOffice
	if code := get(t, "/postoffices/near?lat=-0.2833&lng=36.0667", &offices); code != http.StatusOK || len(offices) != 5 {
		t.Fatalf("status %d, %d offices, want 5", code, len(offices))
	}
	if offices[0].Code != "20100" || offices[0].DistanceKm > 5 {
		t.Errorf("nearest to Nakuru town = %+v, want 20100 Nakuru", offices[0])
	}
	if get(t, "/postoffices/near?lat=-0.2833&lng=36.0667&limit=2", &offices); len(offices) != 2 {
		t.Errorf("limit=2 returned %d offices", len(offices))
	}
}

func TestCountyAt(t *testing.T) {
	var c kenyageo.County
	if code := get(t, "/counties/at?lat=-0.0917&lng=34.768", &c); code != http.StatusOK || c.Code != 42 {
		t.Errorf("status %d, county %d %s, want 42 Kisumu", code, c.Code, c.Name)
	}
}

func TestBoundaries(t *testing.T) {
	for _, path := range []string{"/wards/1439/boundary", "/counties/nairobi/boundary"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d", path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/geo+json" {
			t.Errorf("%s: Content-Type = %q", path, ct)
		}
		var f struct {
			Type     string
			Geometry struct{ Type string }
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &f); err != nil || f.Type != "Feature" || f.Geometry.Type != "MultiPolygon" {
			t.Errorf("%s: %+v, %v", path, f, err)
		}
	}
}

func TestSearch(t *testing.T) {
	var matches []Match
	if code := get(t, "/search?q=nakru&limit=3", &matches); code != http.StatusOK || len(matches) == 0 {
		t.Fatalf("status %d, %d matches", code, len(matches))
	}
	if m := matches[0]; m.Kind != "county" || m.CountyCode != 32 {
		t.Errorf("first match = %+v, want Nakuru county", m)
	}
	if code := get(t, "/search?q=zzzzzzzz", &matches); code != http.StatusOK || len(matches) != 0 {
		t.Errorf("no-match search: status %d, %d matches", code, len(matches))
	}
}

func TestErrors(t *testing.T) {
	tests := map[string]int{
		"/counties/48":                             http.StatusNotFound,
		"/counties/atlantis":                       http.StatusNotFound,
		"/wards/9999":                              http.StatusNotFound,
		"/wards/9999/boundary":                     http.StatusNotFound,
		"/wards/0097/boundary":                     http.StatusNotFound,
		"/counties/48/boundary":                    http.StatusNotFound,
		"/postcodes/99999":                         http.StatusNotFound,
		"/constituencies/nowhere/wards":            http.StatusNotFound,
		"/search":                                  http.StatusBadRequest,
		"/search?q=x&limit=0":                      http.StatusBadRequest,
		"/search?q=x&limit=lots":                   http.StatusBadRequest,
		"/at?lat=north&lng=1":                      http.StatusBadRequest,
		"/at?lat=-4.5&lng=40.5":                    http.StatusNotFound,
		"/counties/at?lat=x&lng=1":                 http.StatusBadRequest,
		"/counties/at?lat=-4.5&lng=40.5":           http.StatusNotFound,
		"/postoffices/near?lat=1":                  http.StatusBadRequest,
		"/postoffices/near?lat=1&lng=37&limit=500": http.StatusBadRequest,
		"/nothing-here":                            http.StatusNotFound,
	}
	for path, want := range tests {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != want {
			t.Errorf("%s: status %d, want %d", path, rec.Code, want)
		}
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/wards/9999", nil))
	var body struct{ Error string }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Error == "" {
		t.Errorf("error body = %q", rec.Body)
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/counties", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /counties: status %d, want 405", rec.Code)
	}
}

func Example() {
	srv := httptest.NewServer(New(kenyageo.Default()))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/at?lat=-1.2884&lng=36.8233")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Print(string(body))
	// Output: {"ward_code":"1439","name":"NAIROBI CENTRAL","constituency_name":"Starehe","county_code":47,"county_name":"Nairobi County","registered_voters_2022":52186,"centroid":{"lat":-1.28685,"lng":36.82961}}
}
