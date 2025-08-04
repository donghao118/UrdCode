package utils

import (
	"strings"
)

type RangeTree struct {
	start, end string
	root       *RangeTreeNode
}

func (t *RangeTree) StartKey() string { return t.start }
func (t *RangeTree) EndKey() string   { return t.end }

func NewRangeTree() *RangeTree {
	return &RangeTree{
		start: "",
		end:   "",
		root:  nil,
	}
}

func (t *RangeTree) Search(p string) bool {
	if t.root == nil {
		return false
	}
	return t.root.Search(p, t.start, t.end)
}

func (t *RangeTree) AddRange(rangeLow, rangeHigh string) error {
	if rangeHigh < rangeLow {
		return nil
	}

	if t.root == nil {
		t.start, t.end = rangeLow, rangeHigh
		t.root = NewRangeTreeLeaf(t.end)
		return nil
	}
	if rangeHigh < t.start {
		rightNode := t.root
		rightNode.Ukey = t.start
		leftNode := NewRangeTreeLeaf(rangeHigh)
		t.start = rangeLow
		t.root = &RangeTreeNode{
			Ukey:       t.end,
			LeftChild:  leftNode,
			RightChild: rightNode,
			depth:      rightNode.depth + 1,
		}
		return nil
	}
	if rangeLow > t.end {
		leftNode := t.root
		rightNode := NewRangeTreeLeaf(rangeLow)
		t.end = rangeHigh
		t.root = &RangeTreeNode{
			Ukey:       t.end,
			LeftChild:  leftNode,
			RightChild: rightNode,
			depth:      leftNode.depth + 1,
		}
		return nil
	}
	if rangeHigh > t.end {
		t.end = rangeHigh
		t.root.Ukey = t.end
	}
	if rangeLow < t.start {
		t.start = rangeLow
	}
	t.root.AddRange(t.start, rangeLow, rangeHigh, t.end)
	return nil
}

func (t *RangeTree) DeleteRange(rangeLow, rangeHigh string) error {
	if rangeHigh < rangeLow || t.root == nil {
		return nil
	}
	if rangeHigh < t.start || rangeLow > t.end {
		return nil
	}
	if rangeLow < t.start {
		rangeLow = t.start
	}
	if rangeHigh > t.end {
		rangeHigh = t.end
	}
	t.root, t.start, t.end = t.root.DeleteRange(t.start, rangeLow, rangeHigh, t.end)
	return nil
}

func (t *RangeTree) Range() [][]string {
	if t.root == nil {
		return nil
	}
	return t.root.Range(nil, t.start, t.end)
}

func (t *RangeTree) Add(other *RangeTree) {
	for _, pair := range other.Range() {
		t.AddRange(pair[0], pair[1])
	}
}


func NewRangeTreeFromString(s string) *RangeTree {
	t := NewRangeTree()
	for _, pair := range deserialize(s) {
		t.AddRange(pair[0], pair[1])
	}
	return t
}

func deserialize(s string) [][]string {
	subStrings := strings.Split(s, "+")
	result := make([][]string, len(subStrings))
	for i, subString := range subStrings {
		values := strings.Split(subString[1:len(subString)-1], ",")
		result[i] = values
	}
	return result
}

type RangeTreeNode struct {
	Ukey       string
	LeftChild  *RangeTreeNode
	RightChild *RangeTreeNode
	depth      int
}

func (t *RangeTreeNode) IsLeaf() bool {
	return t.LeftChild == nil
}

func (t *RangeTreeNode) AddRange(low, rangeLow, rangeHigh, high string) {
	if t.IsLeaf() {
		return
	}
	defer func() {
		if t.IsLeaf() {
			t.depth = 0
			return
		}
		if t.LeftChild.IsLeaf() && t.RightChild.IsLeaf() && t.LeftChild.Ukey == t.RightChild.Ukey {
			t.LeftChild, t.RightChild = nil, nil
			t.depth = 0
			return
		}
		ld := t.LeftChild.depth
		rd := t.RightChild.depth
		if ld > rd {
			t.depth = ld + 1
		} else {
			t.depth = rd + 1
		}
	}()
	//case 1: [low   *rl   )left  [right   *rh   )high
	if strInRange(rangeLow, low, t.LeftChild.Ukey) && strInRange(rangeHigh, t.RightChild.Ukey, high) {
		t.LeftChild.AddRange(low, rangeLow, t.LeftChild.Ukey, t.LeftChild.Ukey)
		t.RightChild.AddRange(t.RightChild.Ukey, t.RightChild.Ukey, rangeHigh, high)
		if t.LeftChild.depth < t.RightChild.depth {
			t.LeftChild.Ukey = t.RightChild.Ukey
		} else {
			t.RightChild.Ukey = t.LeftChild.Ukey
		}
		return
	}
	//case 2:[low  *rl   *rh  )left  [right   )high
	if strInRange(rangeLow, low, t.LeftChild.Ukey) && strInRange(rangeHigh, low, t.LeftChild.Ukey) {
		t.LeftChild.AddRange(low, rangeLow, rangeHigh, t.LeftChild.Ukey)
		return
	}
	//case 3:[low   )left  [right  *rl   *rh  )high
	if strInRange(rangeLow, t.RightChild.Ukey, high) && strInRange(rangeHigh, t.RightChild.Ukey, high) {
		t.RightChild.AddRange(t.RightChild.Ukey, rangeLow, rangeHigh, high)
		return
	}
	//case 4:[low  )left   *rl   [right  *rh  )high
	if strInRange(rangeLow, t.LeftChild.Ukey, t.RightChild.Ukey) && strInRange(rangeHigh, t.RightChild.Ukey, high) {
		t.RightChild.Ukey = rangeLow
		t.RightChild.AddRange(rangeLow, rangeLow, rangeHigh, high)
		return
	}
	//case 5:[low  *rl )left   *rh  [right   )high
	if strInRange(rangeLow, low, t.LeftChild.Ukey) && strInRange(rangeHigh, t.LeftChild.Ukey, t.RightChild.Ukey) {
		t.LeftChild.Ukey = rangeHigh
		t.LeftChild.AddRange(low, rangeLow, rangeHigh, rangeHigh)
		return
	}

	//case 6:[low  )left  *rl  *rh  [right   )high
	if strInRange(rangeLow, t.LeftChild.Ukey, t.RightChild.Ukey) && strInRange(rangeHigh, t.LeftChild.Ukey, t.RightChild.Ukey) {
		if t.LeftChild.depth < t.RightChild.depth {
			newRight := NewRangeTreeLeaf(rangeLow)
			newNode := &RangeTreeNode{
				Ukey:       rangeHigh,
				LeftChild:  t.LeftChild,
				RightChild: newRight,
				depth:      t.LeftChild.depth + 1,
			}
			t.LeftChild = newNode
		} else {
			newLeft := NewRangeTreeLeaf(rangeHigh)
			newNode := &RangeTreeNode{
				Ukey:       rangeLow,
				LeftChild:  newLeft,
				RightChild: t.RightChild,
				depth:      t.RightChild.depth + 1,
			}
			t.RightChild = newNode
		}
		return
	}
	panic("RangeTreeNode cannot add a range beyond the [low,high] boundary@")
}

func (t *RangeTreeNode) DeleteRange(low, rangeLow, rangeHigh, high string) (*RangeTreeNode, string, string) {
	if rangeLow == low && rangeHigh == high {
		return nil, low, high
	}
	if rangeLow == rangeHigh {
		return t, low, high
	}
	if t.IsLeaf() {
		if rangeHigh == high {
			if t.Ukey == high {
				t.Ukey = rangeLow
			}
			return t, low, rangeLow
		}
		if rangeLow == low {
			if t.Ukey == low {
				t.Ukey = rangeHigh
			}
			return t, rangeHigh, high
		}
		t.LeftChild = NewRangeTreeLeaf(rangeLow)
		t.RightChild = NewRangeTreeLeaf(rangeHigh)
		t.depth++
		return t, low, high
	}
	var leftRange, rightRange string = low, high

	switch {
	//case 1: [low   *rl   )left  [right   *rh   )high
	case strInRange(rangeLow, low, t.LeftChild.Ukey) && strInRange(rangeHigh, t.RightChild.Ukey, high):
		t.LeftChild, leftRange, _ = t.LeftChild.DeleteRange(low, rangeLow, t.LeftChild.Ukey, t.LeftChild.Ukey)
		t.RightChild, _, rightRange = t.RightChild.DeleteRange(t.RightChild.Ukey, t.RightChild.Ukey, rangeHigh, high)

	//case 2:[low  *rl   *rh  )left  [right   )high
	case strInRange(rangeLow, low, t.LeftChild.Ukey) && strInRange(rangeHigh, low, t.LeftChild.Ukey):
		t.LeftChild, leftRange, _ = t.LeftChild.DeleteRange(low, rangeLow, rangeHigh, t.LeftChild.Ukey)

	//case 3:[low   )left  [right  *rl   *rh  )high
	case strInRange(rangeLow, t.RightChild.Ukey, high) && strInRange(rangeHigh, t.RightChild.Ukey, high):
		t.RightChild, _, rightRange = t.RightChild.DeleteRange(t.RightChild.Ukey, rangeLow, rangeHigh, high)

	//case 4:[low  )left   *rl   [right  *rh  )high
	case strInRange(rangeLow, t.LeftChild.Ukey, t.RightChild.Ukey) && strInRange(rangeHigh, t.RightChild.Ukey, high):
		t.RightChild, _, rightRange = t.RightChild.DeleteRange(t.RightChild.Ukey, t.RightChild.Ukey, rangeHigh, high)

	//case 5:[low  *rl )left   *rh  [right   )high
	case strInRange(rangeLow, low, t.LeftChild.Ukey) && strInRange(rangeHigh, t.LeftChild.Ukey, t.RightChild.Ukey):
		t.LeftChild, leftRange, _ = t.LeftChild.DeleteRange(low, rangeLow, t.LeftChild.Ukey, t.LeftChild.Ukey)

	// case 6:[low  )left  *rl  *rh  [right   )high
	case strInRange(rangeLow, t.LeftChild.Ukey, t.RightChild.Ukey) && strInRange(rangeHigh, t.LeftChild.Ukey, t.RightChild.Ukey):
	}

	if t.LeftChild == nil && t.RightChild == nil {
		return nil, leftRange, rightRange
	}

	var IsLeft bool
	if t.Ukey == high {
		t.Ukey = rightRange
		IsLeft = true
	} else if t.Ukey == low {
		t.Ukey = leftRange
		IsLeft = false
	} else {
		panic("t.Ukey is not one of the bounds, no such node exists")
	}

	if t.LeftChild != nil && t.RightChild != nil {
		if t.LeftChild.depth > t.RightChild.depth {
			t.depth = t.LeftChild.depth + 1
		} else {
			t.depth = t.RightChild.depth + 1
		}
		return t, leftRange, rightRange
	}

	if t.LeftChild != nil {
		if IsLeft {
			return t.LeftChild, leftRange, t.LeftChild.Ukey
		}

		rightRange = t.LeftChild.Ukey
		t.LeftChild.Ukey = leftRange
		return t.LeftChild, leftRange, rightRange
	} else {
		if IsLeft {

			leftRange = t.RightChild.Ukey
			t.RightChild.Ukey = rightRange
			return t.RightChild, leftRange, rightRange
		}
		return t.RightChild, t.RightChild.Ukey, rightRange
	}
}

func (t *RangeTreeNode) Search(p string, low, high string) bool {
	if t.IsLeaf() {
		return strInRangeRightOpen(p, low, high)
	}
	if p < t.LeftChild.Ukey {
		return t.LeftChild.Search(p, low, t.LeftChild.Ukey)
	} else if p >= t.RightChild.Ukey {
		return t.RightChild.Search(p, t.RightChild.Ukey, high)
	}
	return false
}
func (t *RangeTreeNode) Range(outputList [][]string, start, end string) [][]string {
	if t.IsLeaf() {
		outputList = append(outputList, []string{start, end})
		return outputList
	}
	outputList = t.LeftChild.Range(outputList, start, t.LeftChild.Ukey)
	outputList = t.RightChild.Range(outputList, t.RightChild.Ukey, end)
	return outputList
}

func strInRange(s, u, v string) bool {
	return u <= s && s <= v
}

func strInRangeRightOpen(s, u, v string) bool {
	return u <= s && s < v
}

func NewRangeTreeLeaf(ukey string) *RangeTreeNode {
	return &RangeTreeNode{
		Ukey:       ukey,
		LeftChild:  nil,
		RightChild: nil,
		depth:      0,
	}
}
