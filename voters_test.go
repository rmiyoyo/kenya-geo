package kenyageo

import (
	"fmt"
	"testing"
)

func TestNationalVoters(t *testing.T) {
	n := Default().NationalVoters()
	if n.Wards != 1450 || n.Complete || n.RegisteredVoters2022 < 22_000_000 || n.RegisteredVoters2022 > 22_200_000 {
		t.Errorf("national total = %+v", n)
	}
}

func TestVotersAddUp(t *testing.T) {
	d := Default()
	sumCounties, sumConsts := 0, 0
	for _, c := range d.Counties() {
		ct, ok := d.CountyVoters(c.Code)
		if !ok {
			t.Fatalf("no voters for county %d", c.Code)
		}
		sumCounties += ct.RegisteredVoters2022
		inCounty := 0
		for _, name := range d.Constituencies(c.Code) {
			v, ok := d.ConstituencyVoters(name)
			if !ok {
				t.Fatalf("no voters for constituency %s", name)
			}
			inCounty += v.RegisteredVoters2022
		}
		if inCounty != ct.RegisteredVoters2022 {
			t.Errorf("%s: constituencies add up to %d, county total is %d", c.Name, inCounty, ct.RegisteredVoters2022)
		}
		sumConsts += inCounty
	}
	if n := d.NationalVoters().RegisteredVoters2022; sumCounties != n || sumConsts != n {
		t.Errorf("counties %d, constituencies %d, national %d", sumCounties, sumConsts, n)
	}
}

func TestVotersIncomplete(t *testing.T) {
	d := Default()
	if v, _ := d.ConstituencyVoters("Eldas"); v.Complete {
		t.Errorf("Eldas = %+v, want incomplete because Della has no figure", v)
	}
	if v, _ := d.CountyVoters(8); v.Complete {
		t.Errorf("Wajir = %+v, want incomplete", v)
	}
	if v, _ := d.CountyVoters(47); !v.Complete || v.Wards != 85 {
		t.Errorf("Nairobi = %+v", v)
	}
	if _, ok := d.CountyVoters(48); ok {
		t.Error("found voters for county 48")
	}
	if _, ok := d.ConstituencyVoters("nowhere"); ok {
		t.Error("found voters for constituency nowhere")
	}
}

func ExampleData_CountyVoters() {
	v, _ := Default().CountyVoters(47)
	fmt.Println(v.RegisteredVoters2022, v.Wards, v.Complete)
	// Output: 2415310 85 true
}
