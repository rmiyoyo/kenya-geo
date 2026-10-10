package kenyageo

type VoterTotal struct {
	RegisteredVoters2022 int  `json:"registered_voters_2022"`
	Wards                int  `json:"wards"`
	Complete             bool `json:"complete"`
}

func (d *Data) NationalVoters() VoterTotal {
	t := VoterTotal{Complete: true}
	for _, w := range d.wards {
		t.add(w)
	}
	return t
}

func (d *Data) CountyVoters(code int) (VoterTotal, bool) {
	return d.sumVoters(d.wardsByCounty[code])
}

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
