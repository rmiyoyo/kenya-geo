package kenyageo_test

import (
	"errors"
	"fmt"

	kenyageo "github.com/rmiyoyo/kenya-geo"
)

func Example() {
	geo := kenyageo.Default()
	nakuru, _ := geo.CountyByName("nakuru")
	fmt.Println(nakuru.Code, nakuru.Name, nakuru.Population)
	// Output:
	// 32 Nakuru County 2162202
}

func ExampleData_WardAt() {
	geo := kenyageo.Default()
	ward, ok := geo.WardAt(-1.2884, 36.8233)
	fmt.Println(ok, ward.Name, ward.Constituency, ward.CountyName)
	// Output:
	// true NAIROBI CENTRAL Starehe Nairobi County
}

func ExampleData_CountyAt() {
	geo := kenyageo.Default()
	county, _ := geo.CountyAt(-0.0917, 34.768)
	fmt.Println(county.Name)
	// Output:
	// Kisumu County
}

func ExampleData_Search() {
	geo := kenyageo.Default()
	for _, m := range geo.Search("nakru", 3) {
		fmt.Println(m.Kind, m.Name, m.Code)
	}
	// Output:
	// county Nakuru County 32
	// post office Nakuru 20100
}

func ExampleData_ParseAddress() {
	geo := kenyageo.Default()
	a, err := geo.ParseAddress("p.o. box 123, nairobi 00100")
	fmt.Println(a, a.PostOffice.Name, err)

	_, err = geo.ParseAddress("P.O. Box 90-40100 Eldoret")
	fmt.Println(errors.Is(err, kenyageo.ErrTownMismatch))
	// Output:
	// P.O. Box 123-00100 Nairobi Nairobi Gpo <nil>
	// true
}

func ExampleData_PostOfficesNear() {
	geo := kenyageo.Default()
	for _, p := range geo.PostOfficesNear(-0.2833, 36.0667, 2) {
		fmt.Printf("%s %s %.1f km\n", p.Code, p.Name, p.DistanceKm)
	}
	// Output:
	// 20100 Nakuru 2.7 km
	// 20112 Lanet 8.2 km
}

func ExampleData_CountyVoters() {
	geo := kenyageo.Default()
	v, _ := geo.CountyVoters(47)
	fmt.Println(v.RegisteredVoters2022, v.Wards, v.Complete)
	// Output:
	// 2415310 85 true
}

func ExampleData_National() {
	geo := kenyageo.Default()
	kenya, _ := geo.National()
	fmt.Printf("%.1f%% Christian, %.1f%% of homes on mains electricity\n",
		kenya.Religion.Pct(kenya.Religion.Christian()), kenya.Living.ElectricityPct)
	// Output:
	// 85.5% Christian, 50.4% of homes on mains electricity
}

func ExampleCounty_UrbanPct() {
	geo := kenyageo.Default()
	kilifi, _ := geo.CountyByCode(3)
	fmt.Printf("%.1f%% urban, %.1f%% use the internet\n", kilifi.UrbanPct(), kilifi.Living.InternetPct)
	// Output:
	// 27.1% urban, 15.1% use the internet
}
