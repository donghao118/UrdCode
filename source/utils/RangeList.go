package utils

import "strings"

type RangeStore interface {
	Add(other RangeStore)
	AddRange(rangeLow string, rangeHigh string) error
	DeleteRange(rangeLow string, rangeHigh string) error
	EndKey() string
	Range() [][]string
	Search(p string) bool
	StartKey() string
}

var _ RangeStore = (*RangeList)(nil)

type RangeList struct {
	lists []string
}

func NewRangeList() *RangeList { return &RangeList{lists: nil} }
func NewRangeListFromString(s string) *RangeList {
	r := NewRangeList()
	labels := strings.Split(s, ",")
	if len(labels)%2 == 1 {
		return nil
	}
	for i := 0; i < len(labels); i += 2 {
		r.AddRange(labels[i], labels[i+1])
	}
	return r
}

func (r *RangeList) String() string {
	labels := []string{}
	for _, str := range r.Range() {
		labels = append(labels, str...)
	}
	return strings.Join(labels, ",")
}

func (r *RangeList) Add(other RangeStore) {
	for _, rg := range other.Range() {
		if len(rg) == 2 {
			r.AddRange(rg[0], rg[1])
		}
	}
}

func (r *RangeList) StartKey() string {
	if r.lists == nil {
		return ""
	}
	return r.lists[0]
}
func (r *RangeList) EndKey() string {
	if r.lists == nil {
		return ""
	}
	return r.lists[len(r.lists)-1]
}
func (r *RangeList) Range() [][]string {
	out := make([][]string, len(r.lists)/2)
	for i := 0; i < len(r.lists)/2; i++ {
		out[i] = []string{r.lists[i*2], r.lists[i*2+1]}
	}
	return out
}
func (r *RangeList) Search(p string) bool {
	return r.redPlace(r.find(p))
}
func (r *RangeList) AddRange(rangeLow, rangeHigh string) error {
	if rangeHigh <= rangeLow {
		return nil
	}
	if r.nilRagne() {
		r.lists = append(r.lists, rangeLow)
		r.lists = append(r.lists, rangeHigh)
		return nil
	}
	pl, pr := r.find(rangeLow), r.find(rangeHigh)
	if pr == pl && r.redPlace(pl) {
		return nil
	}
	var plList, rangeList, prList []string
	rangeList = make([]string, 2)
	if r.redPlace(pl) {
		plList = r.leftRange(pl)
		rangeList[0] = r.get(pl)
	} else {
		if r.stepThere(rangeLow, pl) {
			plList = r.leftRange(pl - 1)
			rangeList[0] = r.get(pl - 1)
		} else {
			plList = r.leftRange(pl + 1)
			rangeList[0] = rangeLow
		}
	}

	if r.redPlace(pr) {
		prList = r.rightRange(pr + 1)
		rangeList[1] = r.get(pr + 1)
	} else {
		if r.stepThere(rangeHigh, pr) {
			prList = r.rightRange(pr)
			rangeList[1] = r.get(pr)
		} else {
			prList = r.rightRange(pr)
			rangeList[1] = rangeHigh
		}
	}

	nl, nr := len(plList), len(prList)
	new_lists := make([]string, nl+2+nr)
	if nl > 0 {
		copy(new_lists[0:nl], plList)
	}
	copy(new_lists[nl:nl+2], rangeList)
	if nr > 0 {
		copy(new_lists[nl+2:nl+2+nr], prList)
	}
	r.lists = new_lists
	return nil
}

func (r *RangeList) DeleteRange(rangeLow, rangeHigh string) error {
	if rangeHigh <= rangeLow {
		return nil
	}
	if r.nilRagne() {
		return nil
	}
	pl, pr := r.find(rangeLow), r.find(rangeHigh)
	if pr == pl && !r.redPlace(pl) {
		return nil
	}
	var plList, midList, prList []string

	if r.redPlace(pl) {
		if r.stepThere(rangeLow, pl) {
			plList = r.leftRange(pl)
		} else {
			plList = r.leftRange(pl + 1)
			midList = append(midList, rangeLow)
		}
	} else {
		plList = r.leftRange(pl + 1)
	}
	if r.redPlace(pr) {
		if r.stepThere(rangeHigh, pr) {
			prList = r.rightRange(pr - 1)
		} else {
			midList = append(midList, rangeHigh)
			prList = r.rightRange(pr)
		}
	} else {
		prList = r.rightRange(pr)
	}

	nl, nm, nr := len(plList), len(midList), len(prList)
	new_lists := make([]string, nl+nm+nr)
	if nl > 0 {
		copy(new_lists[0:nl], plList)
	}
	if nm > 0 {
		copy(new_lists[nl:nl+nm], midList)
	}
	if nr > 0 {
		copy(new_lists[nl+nm:nl+nm+nr], prList)
	}
	r.lists = new_lists
	return nil
}

func (r *RangeList) nilRagne() bool                   { return len(r.lists) == 0 }
func (r *RangeList) stepThere(s string, plc int) bool { return plc >= 0 && r.lists[plc] == s }
func (r *RangeList) get(plc int) string               { return r.lists[plc] }

func (r *RangeList) goodPlace(s string, plc int) bool {
	return (plc == -1 || r.lists[plc] <= s) &&
		(plc == len(r.lists)-1 || s < r.lists[plc+1])
}

func (r *RangeList) find(s string) int {
	start := -1
	end := len(r.lists) - 1
	plc := (start + end) / 2
	for !r.goodPlace(s, plc) {
		if s < r.get(plc) {
			end = plc - 1
		} else {
			start = plc + 1
		}
		plc = (start + end) / 2
	}
	return plc
}

func (r *RangeList) redPlace(plc int) bool { return plc%2 == 0 }
func (r *RangeList) leftRange(plc int) []string {
	if plc == -1 {
		return nil
	}
	return r.lists[:plc]
}
func (r *RangeList) rightRange(plc int) []string {
	if plc == len(r.lists)-1 {
		return nil
	}
	return r.lists[plc+1:]
}
