package fibheap

import S1 "github.com/Sreevani871/fibheap/node"
import S "github.com/Sreevani871/fibheap"
import "testing"

func BenchmarkInsert(b *testing.B) {
	f := S.NewHeap()
	for i := 0; i < b.N; i++ {
		f.Insert(-9.8)
	}

}

func BenchmarkExtractMin(b *testing.B) {
	f := S.NewHeap()
	for i := 0; i < b.N; i++ {
		f.ExtractMin()
	}

}
func BenchmarkGetMinValue(b *testing.B) {
	f := S.NewHeap()
	for i := 0; i < b.N; i++ {
		f.GetMinValue()
	}

}

func BenchmarkMerge(b *testing.B) {
	f := S.NewHeap()
	f1 := S.NewHeap()
	for i := 0; i < b.N; i++ {
		f.Merge(f1)
	}
}

func BenchmarkDecreaseKey(b *testing.B) {
	f := S.NewHeap()
	f.Insert(30)
	n := new(S1.Node)
	for i := 0; i < b.N; i++ {
		f.DecreaseKey(n, 20)
	}
}

func BenchmarkDelete(b *testing.B) {
	f := S.NewHeap()
	f.Insert(4)
	f1 := new(S1.Node)
	for i := 0; i < b.N; i++ {
		f.Delete(f1)
	}

}
func BenchmarkCount(b *testing.B) {
	f := S.NewHeap()
	for i := 0; i < b.N; i++ {
		f.Count()
	}
}

func BenchmarkIsEmpty(b *testing.B) {
	f := S.NewHeap()
	for i := 0; i < b.N; i++ {
		f.IsEmpty()
	}
}

func BenchmarkString(b *testing.B) {
	f := S.NewHeap()
	for i := 0; i < b.N; i++ {
		f.String()
	}
}

func BenchmarkCountNodes(b *testing.B) {
	f := S.NewHeap()
	for i := 0; i < b.N; i++ {
		f.CountNodes()
	}
}

func BenchmarkCheckHeap(b *testing.B) {
	f := S.NewHeap()
	for i := 0; i < b.N; i++ {
		f.CheckHeap()
	}
}
