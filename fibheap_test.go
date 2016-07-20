package fibheap

import "github.com/Sreevani871/fibheap/node"

import S "github.com/Sreevani871/fibheap"
import "testing"

func TestMap(t *testing.T) {
	ranks := make(map[int]*node.Node)
	if len(ranks) != 0 {
		t.Error("Shouldn't a new array be empty??")
	}

	if ranks[0] != nil {
		t.Error("Shouldn't empty entries to pointers be nil?")
	}
}

func TestInsert_ExtractMin(t *testing.T) {

	f := S.NewHeap()
	f.Insert(1.0)

	f.Insert(2.0)
	f.Insert(3.0)
	f.Insert(-1.0)

	v := f.ExtractMin()

	if v != -1.0 {
		t.Error("Expected", -1.0, "got", v)
	}
	f1 := S.NewHeap()
	e := f1.ExtractMin()
	if e != -1 {
		t.Error("Expected", -1, "got", e)
	}
	fh := S.NewHeap()
	for i := -100000.0; i < 100000; i++ {
		fh.Insert(i)
	}
	for i := -100000.0; i < 100000; i++ {
		val := fh.ExtractMin()
		if val != i {
			t.Error("Expected", i, "Got", val)
		}
	}

}
func TestDecreaseKey(t *testing.T) {
	f := S.NewHeap()
	n1 := f.Insert(1.0)
	f.Insert(2.0)
	f.Insert(-3.0)
	f.Insert(-4.0)
	f.Insert(-4.0)
	f.Insert(4.0)
	d := f.Insert(-6.0)
	n2 := f.Insert(2.0)
	n3 := f.Insert(9.0)
	f.Delete(d)
	v1 := f.DecreaseKey(n2, -2)
	if v1.Value != -2 {
		t.Error("Something went wrong")
	}
	v2 := f.DecreaseKey(n1, 3)
	if v2 != nil {
		t.Error("Something went wrong")
	}
	v3 := f.DecreaseKey(n3, -100)
	if v3.Value != -100 {
		t.Error("Something went wrong")
	}

}

func TestGetMinValue(t *testing.T) {
	f := S.NewHeap()
	f.Insert(1.0)

	f.Insert(2.0)
	f.Insert(3.0)
	f.Insert(-1.0)

	v := f.GetMinValue()
	if v != -1.0 {
		t.Error("Expected", -1.0, "got", v)
	}
	f1 := S.NewHeap()
	v1 := f1.GetMinValue()
	if v1 != -1 {
		t.Error("Cannot get min element from empty heap")
	}
}

func TestMerge(t *testing.T) {

	list1 := []float64{1.23, 6.3, -9.8, 0, -90, 23, 3.4}
	list2 := []float64{3, 4, 4.5, -8}
	mergelist := []float64{-90, -9.8, -8, 0, 1.23, 3, 3.4, 4, 4.5, 6.3}
	f2 := S.NewHeap()
	var l1, l2 []float64
	for _, v := range l1 {
		f2.Insert(v)
	}
	f3 := S.NewHeap()
	for _, v := range l2 {
		f3.Insert(v)
	}
	f2.Merge(f3)
	f := S.NewHeap()
	for _, v := range list1 {
		f.Insert(v)
	}
	f1 := S.NewHeap()
	for _, v := range list2 {
		f1.Insert(v)
	}
	f.Merge(f1)
	for _, v := range mergelist {
		val := f.ExtractMin()
		if val != v {
			t.Error("Expected", v, "Got", val)
		}
	}
}

func TestCountNodes(t *testing.T) {
	h := S.NewHeap()
	n := 100
	for i := 0; i < n; i++ {
		h.Insert(float64(n ^ 2 - i))
	}
	h.CountNodes()
}

func TestPrintTrees(t *testing.T) {
	h := S.NewHeap()
	n := 100
	for i := 0; i < n; i++ {
		h.Insert(float64(n ^ 2 - i))
	}
	h.PrintTrees()
}
func TestConsole(t *testing.T) {
	h := S.NewHeap()
	n := 100000
	for i := 0; i < n; i++ {
		h.Insert(float64(n ^ 2 - i))
	}

	if h.Count() != n {
		t.Error("Random heap not built correctly!")
	}

	if !h.CheckHeap() {
		t.Errorf("Heap corrupted: %d/%d", h.Count(), h.Len())
	}

	for h.Count() != 0 && h.CheckHeap() {
		h.ExtractMin()
	}

	if h.String() != "[Empty Heap]" {
		t.Errorf("\nHeap corrupted: %d/%d;\n%s", h.Count(), h.Len(), h)
	}
}
