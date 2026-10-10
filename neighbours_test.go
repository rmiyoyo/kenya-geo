package kenyageo

import (
	"fmt"
	"slices"
	"testing"
)

func TestNeighbouringCounties(t *testing.T) {
	tests := map[int][]int{
		1:  {2, 3},
		9:  {8},
		47: {16, 22, 34},
	}
	d := Default()
	for code, want := range tests {
		var got []int
		for _, c := range d.NeighbouringCounties(code) {
			got = append(got, c.Code)
		}
		if !slices.Equal(got, want) {
			t.Errorf("neighbours of %d = %v, want %v", code, got, want)
		}
	}
	if got := d.NeighbouringCounties(99); got != nil {
		t.Errorf("neighbours of 99 = %v, want nil", got)
	}
}

func TestNeighboursAreSymmetric(t *testing.T) {
	d := Default()
	for _, c := range d.Counties() {
		if len(c.Neighbours) == 0 {
			t.Errorf("%s has no neighbours", c.Name)
		}
		for _, n := range d.NeighbouringCounties(c.Code) {
			if !slices.Contains(n.Neighbours, c.Code) {
				t.Errorf("%s borders %s, but not the other way round", c.Name, n.Name)
			}
			if n.Code == c.Code {
				t.Errorf("%s borders itself", c.Name)
			}
		}
	}
}

func ExampleData_NeighbouringCounties() {
	for _, c := range Default().NeighbouringCounties(47) {
		fmt.Println(c.Name)
	}
	// Output:
	// Machakos County
	// Kiambu County
	// Kajiado County
}
