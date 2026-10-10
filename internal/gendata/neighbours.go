package main

import (
	"math"
	"sort"
)

const (
	neighbourCell      = 0.01
	neighbourMinPoints = 3
)

func setNeighbours(counties []County, shapes []CountyShape) {
	cells := make(map[int]map[[2]int]bool, len(shapes))
	for _, s := range shapes {
		set := map[[2]int]bool{}
		for _, p := range s.Polygons {
			for _, pt := range p[0] {
				set[cellOf(pt)] = true
			}
		}
		cells[s.Code] = set
	}

	byCode := map[int]*County{}
	for i := range counties {
		byCode[counties[i].Code] = &counties[i]
		counties[i].Neighbours = []int{}
	}
	for _, a := range shapes {
		for _, b := range shapes {
			if a.Code >= b.Code || sharedCells(cells[a.Code], cells[b.Code]) < neighbourMinPoints {
				continue
			}
			byCode[a.Code].Neighbours = append(byCode[a.Code].Neighbours, b.Code)
			byCode[b.Code].Neighbours = append(byCode[b.Code].Neighbours, a.Code)
		}
	}
	for i := range counties {
		sort.Ints(counties[i].Neighbours)
	}
}

func cellOf(pt [2]float64) [2]int {
	return [2]int{int(math.Round(pt[0] / neighbourCell)), int(math.Round(pt[1] / neighbourCell))}
}

func sharedCells(a, b map[[2]int]bool) int {
	n := 0
	for c := range a {
		found := false
		for dx := -1; dx <= 1 && !found; dx++ {
			for dy := -1; dy <= 1 && !found; dy++ {
				found = b[[2]int{c[0] + dx, c[1] + dy}]
			}
		}
		if found {
			n++
		}
	}
	return n
}
