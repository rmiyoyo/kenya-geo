package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	kenyageo "github.com/rmiyoyo/kenya-geo"
)

func runCLI(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return stdout.String(), stderr.String(), code
}

func TestText(t *testing.T) {
	tests := map[string]string{
		"county mombasa":         "641,913 registered voters in 2022",
		"postcode 00100":         "00100  Nairobi Gpo  (Nairobi County)",
		"search kibra":           "constituency",
		"at -1.2884 36.8233":     "1439  NAIROBI CENTRAL ward, Starehe constituency, Nairobi County",
		"near -0.2833,36.0667":   "20100  Nakuru",
		"address Box 45, Kisumu": "P.O. Box 45-40100 Kisumu",
	}
	for args, want := range tests {
		out, errOut, code := runCLI(t, strings.Fields(args)...)
		if code != 0 || !strings.Contains(out, want) {
			t.Errorf("%s: exit %d, stdout %q, stderr %q, want %q", args, code, out, errOut, want)
		}
	}
}

func TestJSON(t *testing.T) {
	out, _, code := runCLI(t, "-json", "county", "mombasa")
	var c countyReport
	if err := json.Unmarshal([]byte(out), &c); err != nil || code != 0 {
		t.Fatalf("exit %d, %v\n%s", code, err, out)
	}
	if c.Code != 1 || c.Voters.RegisteredVoters2022 != 641913 || len(c.Constituencies) != 6 {
		t.Errorf("county = %d, voters %+v, %d constituencies", c.Code, c.Voters, len(c.Constituencies))
	}

	out, _, _ = runCLI(t, "-json", "at", "-1.2884", "36.8233")
	var loc struct {
		Ward   *kenyageo.Ward
		County kenyageo.County
	}
	if err := json.Unmarshal([]byte(out), &loc); err != nil || loc.Ward == nil || loc.Ward.Code != "1439" || loc.County.Code != 47 {
		t.Errorf("at = %+v, %v", loc, err)
	}

	out, _, _ = runCLI(t, "-json", "near", "-0.2833", "36.0667")
	var offices []kenyageo.NearbyPostOffice
	if err := json.Unmarshal([]byte(out), &offices); err != nil || len(offices) != 5 {
		t.Errorf("near = %d offices, %v", len(offices), err)
	}

	out, _, _ = runCLI(t, "-json", "search", "zzzzzzzz")
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("empty search = %q, want []", out)
	}

	out, _, _ = runCLI(t, "boundary", "1439")
	var f struct{ Type string }
	if err := json.Unmarshal([]byte(out), &f); err != nil || f.Type != "Feature" {
		t.Errorf("boundary = %q, %v", f.Type, err)
	}
}

func TestExitCodes(t *testing.T) {
	tests := map[string]int{
		"":                            2,
		"county":                      2,
		"teleport home":               2,
		"-nope county nakuru":         2,
		"at north east":               2,
		"county atlantis":             1,
		"postcode 99999":              1,
		"at -4.5 40.5":                1,
		"address 12 Moi Avenue":       1,
		"address Box 1-00100 Mombasa": 1,
	}
	for args, want := range tests {
		if _, _, code := runCLI(t, strings.Fields(args)...); code != want {
			t.Errorf("%q: exit %d, want %d", args, code, want)
		}
	}
}

func TestMismatchStillPrintsAddress(t *testing.T) {
	out, errOut, _ := runCLI(t, "-json", "address", "Box", "1-00100", "Mombasa")
	var a kenyageo.Address
	if err := json.Unmarshal([]byte(out), &a); err != nil || a.PostalCode != "00100" {
		t.Errorf("address = %+v, %v", a, err)
	}
	if !strings.Contains(errOut, "does not match") {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestThousands(t *testing.T) {
	for n, want := range map[int]string{0: "0", 999: "999", 1000: "1,000", 2415310: "2,415,310"} {
		if got := thousands(n); got != want {
			t.Errorf("thousands(%d) = %q, want %q", n, got, want)
		}
	}
}
