package utils

type DependencyGraph struct {
	dpMap map[string]int
}

func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		dpMap: map[string]int{},
	}
}

func (dg *DependencyGraph) AddWrite(w string, index int) {
	dg.dpMap[w] = index
}
func (dg *DependencyGraph) AddRead(r string) (int, bool) {
	index, ok := dg.dpMap[r]
	return index, ok
}

func (dg *DependencyGraph) Map() map[string]int {
	return dg.dpMap
}
