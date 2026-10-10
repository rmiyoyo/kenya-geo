package kenyageo

import (
	"errors"
	"fmt"
	"testing"
)

func TestParseAddress(t *testing.T) {
	tests := []struct {
		in, box, code, town, office string
		bag                         bool
	}{
		{"P.O. Box 123-00100 Nairobi", "123", "00100", "Nairobi", "Nairobi Gpo", false},
		{"P. O. BOX 1234 - 20100 NAKURU", "1234", "20100", "Nakuru", "Nakuru", false},
		{"Jane Wanjiru\nP.O Box 30100, 00100 GPO Nairobi, Kenya", "30100", "00100", "Nairobi", "Nairobi Gpo", false},
		{"po box 45, 40100 kisumu", "45", "40100", "Kisumu", "Kisumu", false},
		{"Post Office Box 7 80100 Mombasa", "7", "80100", "Mombasa", "Mombasa", false},
		{"P.O. Box 123-00101 Nairobi", "123", "00101", "Nairobi", "Jamia", false},
		{"Box 45, Kisumu", "45", "40100", "Kisumu", "Kisumu", false},
		{"P.O. Box 88, Nakuru 20100", "88", "20100", "Nakuru", "Nakuru", false},
		{"P.O. Box 9-20100 Nakru", "9", "20100", "Nakru", "Nakuru", false},
		{"Private Bag 00100 Nairobi", "", "00100", "Nairobi", "Nairobi Gpo", true},
		{"Private Bag 12-20100 Nakuru", "12", "20100", "Nakuru", "Nakuru", true},
		{"P.O. Box 100-00100", "100", "00100", "", "Nairobi Gpo", false},
	}
	d := Default()
	for _, tt := range tests {
		a, err := d.ParseAddress(tt.in)
		if err != nil {
			t.Errorf("%q: %v", tt.in, err)
			continue
		}
		if a.Box != tt.box || a.PostalCode != tt.code || a.Town != tt.town || a.PrivateBag != tt.bag || a.PostOffice == nil || a.PostOffice.Name != tt.office {
			t.Errorf("%q = %+v %v", tt.in, a, a.PostOffice)
		}
	}
}

func TestParseAddressErrors(t *testing.T) {
	tests := map[string]error{
		"123 Moi Avenue, Nairobi":    ErrNoBox,
		"":                           ErrNoBox,
		"P.O. Box 9-99999 Nowhere":   ErrUnknownPostOffice,
		"P.O. Box 9 Atlantis":        ErrUnknownPostOffice,
		"P.O. Box 9":                 ErrUnknownPostOffice,
		"P.O. Box 123-00100 Mombasa": ErrTownMismatch,
	}
	d := Default()
	for in, want := range tests {
		if _, err := d.ParseAddress(in); !errors.Is(err, want) {
			t.Errorf("%q: err = %v, want %v", in, err, want)
		}
	}
	a, _ := d.ParseAddress("P.O. Box 123-00100 Mombasa")
	if a.PostOffice == nil || a.PostOffice.Code != "00100" || a.Town != "Mombasa" {
		t.Errorf("mismatch still returns the parsed address, got %+v", a)
	}
}

func TestAddressString(t *testing.T) {
	tests := map[string]string{
		"p.o.box 123 - 00100, nairobi": "P.O. Box 123-00100 Nairobi",
		"PRIVATE BAG 00100 NAIROBI":    "Private Bag 00100 Nairobi",
		"Box 45, Kisumu":               "P.O. Box 45-40100 Kisumu",
	}
	for in, want := range tests {
		a, err := Default().ParseAddress(in)
		if got := a.String(); err != nil || got != want {
			t.Errorf("%q: String() = %q, %v, want %q", in, got, err, want)
		}
	}
}

func ExampleData_ParseAddress() {
	a, err := Default().ParseAddress("P.O. Box 30100, 00100 GPO Nairobi")
	fmt.Println(a.Box, a.PostalCode, a.PostOffice.Name, err)
	fmt.Println(a)
	// Output:
	// 30100 00100 Nairobi Gpo <nil>
	// P.O. Box 30100-00100 Nairobi
}
