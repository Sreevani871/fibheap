package fibheap

import "math"
import "github.com/Sreevani871/fibheap/node"
import "fmt"

var (
	NEG_INF = math.Inf(-1)
)

/* A Fibonacci Heap is collection of trees that satisfy the minimum heap
property. It has a theoretical amortized running time better than other heap
structures like the binomial heap. Nodes on the heap are sorted by floating-
point values.*/
type FibHeap struct {
	trees     *node.NodeList
	min       *node.Node
	nodeCount int
}

// NewHeap returns a pointer to an empty Fibonacci heap.
func NewHeap() *FibHeap {
	f := new(FibHeap)
	f.trees = node.NewNodeList()
	return f
}

// Len returns the number of top-level trees on the heap.
func (f *FibHeap) Len() int {
	return f.trees.Len()
}

// Insert adds a new node to the heap with the given floating-point value. The
// new node is added as a new top-level tree. Values may range from -Inf to Inf.
func (f *FibHeap) Insert(v float64) *node.Node {
	if v == NEG_INF {
		fmt.Sprintf("Cannot add a value of negative infinity to the heap. %d", f.nodeCount)
	}

	n := node.NewNode(v, nil)
	f.trees.Insert(n)

	if f.min == nil || n.Value < f.min.Value {
		f.min = n
	}
	f.nodeCount++
	//f.consolidate()
	return n
}

/* Merge combines two heaps by inserting the top-level nodes of the second into
the top level of the first. The minimum element is updated accordingly. */
func (f *FibHeap) Merge(g *FibHeap) {
	if g == nil || g.Len() == 0 {
		return
	}

	f.trees.Merge(g.trees)
	if g.min.Value < f.min.Value {
		f.min = g.min
	}
	f.nodeCount += g.nodeCount
}

/* GetMinValue allows a "peek" at the minimum value of the heap. It does not
change the heap or the minimum value at all. If the heap is empty, the heap
throws a runtime panic.*/
func (f *FibHeap) GetMinValue() float64 {
	if f.min == nil {
		return -1
		//panic("Cannot get the min value of an empty heap!")
	}
	return f.min.Value
}

/* ExtractMin extracts the node with the lowest value off the heap and returns the
value. The minimum node's children are added as top-level trees to the heap.
The heap is then consolidated and a new minimum is found. If the heap is
empty, the heap throws a runtime panic. */
func (f *FibHeap) ExtractMin() float64 {
	if f.min == nil || f.Len() == 0 {
		return -1
	}

	min := f.min.Value
	f.trees.Remove(f.min)
	f.nodeCount--

	for c := f.min.Children.Front(); c != nil; c = c.Next {
		c.Parent = nil
		c.Marked = false
	}

	f.trees.Merge(f.min.Children)

	f.consolidate()

	f.resetMin()

	return min
}

/* Consolidate reduces the number of top-level trees by combining trees with the
same degree, or number of children. When consolidate is finished, each top-
level node on the heap has a different degree. Consolidate has no effect on
empty heaps. */
func (f *FibHeap) consolidate() {
	if f.nodeCount == 0 || f.trees.Front() == nil {
		return
	}

	ranks := make(map[int]*node.Node)

	for curr := f.trees.Front(); curr != nil; {
		deg := curr.Degree()

		if ranks[deg] == nil {
			ranks[deg] = curr
			curr = curr.Next
			continue
		}
		if ranks[deg] == curr {
			curr = curr.Next
			continue
		}

		for ranks[deg] != nil {
			rank := ranks[deg]
			ranks[deg] = nil
			if curr.Value <= rank.Value {
				f.trees.Remove(rank)
				curr.AddChild(rank)
			} else {
				f.trees.Remove(curr)
				rank.AddChild(curr)
				curr = rank
			}
		}
	}
}

/* resetMin searches for the minimum node on the top level and sets it to the
heap's min pointer. If the heap is empty, the min pointer is set to nil. */
func (f *FibHeap) resetMin() {
	m := math.Inf(1)
	f.min = nil
	for curr := f.trees.Front(); curr != nil; curr = curr.Next {
		if curr.Value < m {
			f.min = curr
			m = curr.Value
		}
	}
}

/* DecreaseKey takes the given node and reduces its value to the given value.
If the value is greater than the current value of the node, a runtime panic
is thrown. If the new value of the node breaks the minimum heap property,
the node is cut from its parent and added to the top level of the tree. If
its parent is already marked, its parent is also cut from the tree. The cuts
go up the tree until an unmarked node or a root node are reached. If the
parent isn't marked, it is marked. Root nodes can never be marked.*/
func (f *FibHeap) DecreaseKey(n *node.Node, v float64) *node.Node {
	if v > n.Value {
		return nil
		panic("New value is greater than current value.")
	}

	n.Value = v

	if !n.IsRoot() {
		p := n.Parent
		if n.Value < p.Value {
			f.cut(n)
			f.recCut(p)
		}
	}
	f.resetMin()

	return n
}

/* Cut removes the given node from its parent and adds it to the top level of
the heap. It is unmarked in the process, because root nodes cannot be marked.*/
func (f *FibHeap) cut(n *node.Node) {
	p := n.Parent

	n.Parent = nil
	n.Marked = false

	p.Children.Remove(n)
	f.trees.Insert(n)
}

/* recCut uses a recursive cutting method to remove all marked nodes from the
tree and add them to the heap's top level. If it finds an unmarked node on
the chain up to the root node, that node is marked and the recursive
execution ends.*/
func (f *FibHeap) recCut(n *node.Node) {
	if !n.IsRoot() {
		if n.Marked {
			p := n.Parent
			f.cut(n)
			f.recCut(p)
		} else {
			n.Marked = true
		}
	}
}

/* Delete removes a node from the heap by setting its value to negative infinity,
(which is guaranteed to be the lowest value on the heap. The minimum value
of the heap is then popped, effectively deleting the desired node.*/
func (f *FibHeap) Delete(n *node.Node) {
	f.DecreaseKey(n, NEG_INF)
	f.ExtractMin()
}

// IsEmpty confirms whether there are 0 nodes on the heap.
func (f *FibHeap) IsEmpty() bool {
	return f.nodeCount == 0
}

/* String generates a string representation of the heap. If the heap is empty,
String returns "[Empty Heap]". Otherwise, it returns a recursive string
 representation of the all of the trees on the heap.*/
func (f *FibHeap) String() string {
	if f.Len() == 0 {
		return "[Empty Heap]"
	}

	return f.trees.String()
}

/* Count() return no.of nodes in fibheap */
func (f *FibHeap) Count() int {
	return f.nodeCount
}

/* CheckHeap() returns whether no.of nodes in fibheap and size is equal or not */

func (f *FibHeap) CheckHeap() bool {
	hsize := f.Count()

	count := 0
	for c := f.trees.Front(); c != nil; c = c.Next {
		count += childCount(c.Children)
	}
	if count != hsize {
		return false
	}
	return true
}

func childCount(cl *node.NodeList) int {
	count := 1
	for c := cl.Front(); c != nil; c = c.Next {
		count += childCount(c.Children)
	}
	return count
}

/* CountNodes returns the number of nodes in fibnocci heap*/
func (f *FibHeap) CountNodes() int {
	count := 0
	for c := f.trees.Front(); c != nil; c = c.Next {
		count += childCount(c.Children)
	}
	return count
}

func (f *FibHeap) PrintTrees() {
	f.trees.PrintTree()
}
