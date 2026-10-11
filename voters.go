package kenyageo

// VoterTotal adds up the 2022 registered voters in a group of wards.
// Complete is false when one or more wards have no published figure, so
// the total is a lower bound.
type VoterTotal struct {
	RegisteredVoters2022 int  `json:"registered_voters_2022"`
	Wards                int  `json:"wards"`
	Complete             bool `json:"complete"`
}

// NationalVoters returns the registered voters in every ward.
func (d *Data) NationalVoters() VoterTotal {
	t := VoterTotal{Complete: true}
	for _, w := range d.wards {
		t.add(w)
	}
	return t
}

// CountyVoters returns the registered voters in a county's wards.
func (d *Data) CountyVoters(code int) (VoterTotal, bool) {
	return d.sumVoters(d.wardsByCounty[code])
}

// ConstituencyVoters returns the registered voters in a constituency's
// wards, matching the name like [Data.CountyByName].
func (d *Data) ConstituencyVoters(name string) (VoterTotal, bool) {
	return d.sumVoters(d.wardsByConst[normalize(name)])
}

func (d *Data) sumVoters(idx []int) (VoterTotal, bool) {
	if len(idx) == 0 {
		return VoterTotal{}, false
	}
	t := VoterTotal{Complete: true}
	for _, i := range idx {
		t.add(d.wards[i])
	}
	return t, true
}

func (t *VoterTotal) add(w Ward) {
	t.Wards++
	if w.RegisteredVoters2022 == nil {
		t.Complete = false
		return
	}
	t.RegisteredVoters2022 += *w.RegisteredVoters2022
}
