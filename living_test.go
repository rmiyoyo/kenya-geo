package kenyageo

import "testing"

func TestLiving(t *testing.T) {
	d := Default()
	national, ok := d.NationalLiving()
	if !ok {
		t.Fatal("no national figures")
	}
	want := Living{Households: 12143913, HouseholdSize: 3.9, ElectricityPct: 50.4, InternetPct: 22.6, MobilePhonePct: 47.3}
	if national.Households != want.Households || national.HouseholdSize != want.HouseholdSize ||
		national.ElectricityPct != want.ElectricityPct || national.InternetPct != want.InternetPct ||
		national.MobilePhonePct != want.MobilePhonePct {
		t.Errorf("national %+v, want the KNBS figures %+v", national, want)
	}

	households := 0
	for _, c := range d.Counties() {
		l := c.Living
		households += l.Households
		if l.Households == 0 || l.HouseholdSize < 2 || l.HouseholdSize > 8 {
			t.Errorf("%s: households %d, size %v", c.Name, l.Households, l.HouseholdSize)
		}
		for name, v := range map[string]float64{
			"electricity": l.ElectricityPct, "piped water": l.PipedWaterPct, "internet": l.InternetPct,
			"mobile phone": l.MobilePhonePct, "school attendance": l.SchoolAttendancePct,
		} {
			if v <= 0 || v > 100 {
				t.Errorf("%s: %s %v%% is out of range", c.Name, name, v)
			}
		}
	}
	if households != national.Households {
		t.Errorf("county households add up to %d, national total is %d", households, national.Households)
	}

	nairobi, _ := d.CountyByCode(47)
	if nairobi.Living.Households != 1506888 || nairobi.Living.HouseholdSize != 2.9 || nairobi.Living.ElectricityPct != 96.5 {
		t.Errorf("Nairobi %+v", nairobi.Living)
	}
}

func TestLoadedDataHasNoNationalFigures(t *testing.T) {
	d := &Data{}
	if _, ok := d.NationalLiving(); ok {
		t.Error("NationalLiving reported figures for data without them")
	}
}

func TestUrbanRuralAndReligion(t *testing.T) {
	d := Default()
	national, ok := d.National()
	if !ok {
		t.Fatal("no national figures")
	}
	if national.PopulationUrban != 14831700 || national.PopulationRural != 32732596 {
		t.Errorf("national urban %d, rural %d", national.PopulationUrban, national.PopulationRural)
	}
	r := national.Religion
	if got := r.Pct(r.Christian()); got < 85.4 || got > 85.6 {
		t.Errorf("Kenya is %.2f%% Christian, KNBS says 85.5%%", got)
	}
	if got := r.Pct(r.Islam); got < 10.8 || got > 11.0 {
		t.Errorf("Kenya is %.2f%% Muslim, KNBS says 10.9%%", got)
	}

	urban := 0
	for _, c := range d.Counties() {
		if c.PopulationUrban+c.PopulationRural != c.Population {
			t.Errorf("%s: urban %d + rural %d != population %d", c.Name, c.PopulationUrban, c.PopulationRural, c.Population)
		}
		urban += c.PopulationUrban
		g := c.Religion
		sum := g.Christian() + g.Islam + g.Hindu + g.Traditionist + g.OtherReligion + g.NoReligion + g.DontKnow + g.NotStated
		if g.Total == 0 || sum != g.Total {
			t.Errorf("%s: religious groups add up to %d, total %d", c.Name, sum, g.Total)
		}
	}
	if urban != national.PopulationUrban {
		t.Errorf("county urban populations add up to %d, national is %d", urban, national.PopulationUrban)
	}

	nairobi, _ := d.CountyByCode(47)
	if nairobi.UrbanPct() != 100 {
		t.Errorf("Nairobi is %.1f%% urban", nairobi.UrbanPct())
	}
	mandera, _ := d.CountyByCode(9)
	if mandera.Religion.Pct(mandera.Religion.Islam) < 99 {
		t.Errorf("Mandera %+v", mandera.Religion)
	}
}
