package fibheap

import "github.com/Sreevani871/fibheap/node"

type FibnocciHeap interface {
	Insert(float64) *node.Node
	Merge(*FibHeap)
	ExtractMin() float64
	GetMinimumValue() float64
	DecreasKey(*node.Node, float64)
}
