package node

import "testing"
import S "github.com/Sreevani871/fibheap/node"

//import "fmt"

func BenchmarkNewNode(b *testing.B) {
	f := new(S.Node)
	for i := 0; i < b.N; i++ {
		S.NewNode(6.5, f.Parent)
	}
}

func BenchmarkAddChild(b *testing.B) {
	nl := S.NewNodeList()

	n := S.NewNode(10.0, nil)
	n1 := S.NewNode(29, nil)
	nl.Insert(n)
	for i := 0; i < b.N; i++ {
		n.AddChild(n1)
	}
}

func BenchmarkDeleteChild(b *testing.B) {
	nl := S.NewNodeList()
	n := S.NewNode(10.0, nil)
	n1 := S.NewNode(29, nil)
	nl.Insert(n)
	for i := 0; i < b.N; i++ {
		n.DeleteChild(n1)
	}
}

func BenchmarkFront(b *testing.B) {
	nl := S.NewNodeList()
	for i := 0; i < b.N; i++ {
		nl.Front()
	}

}

func BenchmarkBack(b *testing.B) {
	nl := S.NewNodeList()
	for i := 0; i < b.N; i++ {
		nl.Back()
	}

}

func BenchmarkLen(b *testing.B) {
	nl := S.NewNodeList()
	for i := 0; i < b.N; i++ {
		nl.Len()
	}

}

func BenchmarkRemove(b *testing.B) {
	nl := S.NewNodeList()
	n := S.NewNode(10.0, nil)
	n1 := S.NewNode(29, nil)
	nl.Insert(n)
	nl.Insert(n1)
	for i := 0; i < b.N; i++ {
		nl.Remove(n1)
	}

}

func BenchmarkMerge(b *testing.B) {
	nl := S.NewNodeList()
	n := S.NewNode(10.0, nil)
	n1 := S.NewNode(29, nil)
	nl.Insert(n)
	nl.Insert(n1)
	nl1 := S.NewNodeList()
	n2 := S.NewNode(-9.3, nil)
	n3 := S.NewNode(-1000, nil)
	nl1.Insert(n2)
	nl1.Insert(n3)
	for i := 0; i < b.N; i++ {
		nl.Merge(nl1)
	}

}
