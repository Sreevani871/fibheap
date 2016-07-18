/* Node level operations listed here */

package node

import "fmt"

/* Node is used to build a Fibonacci Heap. It has a non-unique Value that is
used to sort it in the heap. Children is its list of child nodes, and parent
oints to the node that has it as its child. Marked is a flag that indicates
that the node should be cut when flattening the heap. Next is used to
structure a NodeList (linked list) using these nodes. */
type Node struct {
	Value      float64
	Children   *NodeList
	Parent     *Node
	Marked     bool
	Next, prev *Node
}

/* NewNode creates a pointer to a Node object. The value and parent can be set,
while the other values are set to default values.*/
func NewNode(v float64, parent *Node) *Node {
	n := new(Node)
	n.Value = v
	n.Children = NewNodeList()
	n.Parent = parent
	n.Marked = false
	n.Next = nil
	n.prev = nil
	return n
}

// AddChild pushes a child node into this node's list of children.
func (n *Node) AddChild(c *Node) {
	c.Parent = n
	c.Next = nil
	c.prev = nil
	n.Children.Insert(c)
}

/* DeleteChild removes a child from this node's list of children. The child
node's Parent pointer is unset. */
func (n *Node) DeleteChild(c *Node) {
	c.Parent = nil
	n.Children.Remove(c)
}

// Degree returns the number of children that this node has.
func (n *Node) Degree() int {
	return n.Children.Len()
}

/* IsRoot is a predicate function that returns whether or not the Node is at
the root level. This is determined by the state of the node's Parent pointer. */
func (n *Node) IsRoot() bool {
	return n.Parent == nil
}

// String generates the string representation of the node. If the node has no
// children, the result is "(Value)". If the node has children, then the result
// is "(Value: (child)[,(child)])". The string recurses through all of the
// children and builds a representation of the full tree with this node as the
// root.
func (n *Node) String() string {
	if n.Children.Len() == 0 {
		fmt.Sprintf("(%.2f)", n.Value)
		return fmt.Sprintf("(%.2f)", n.Value)
	}
	fmt.Sprintf("(%.2f: %s)", n.Value, n.Children)
	return fmt.Sprintf("(%.2f: %s)", n.Value, n.Children)
}

/* NodeList represents a doubly-linked list of Nodes. It tracks the front and
 back of the list and the list length. The pointer to the previous node is
 hidden from the rest of the program, making this effectively a singly-linked
list. */
type NodeList struct {
	front, back *Node
	length      int
}

// NewNodeList returns a new NodeList head. The list is empty.
func NewNodeList() *NodeList {
	return new(NodeList)
}

/* Front returns the front element pointer of the node list. If the list is
empty, this function returns nil. */
func (nl *NodeList) Front() *Node {
	return nl.front
}

/* Back returns the pointer to the last element of the node list. If the list is
empty, this function returns nil. */
func (nl *NodeList) Back() *Node {
	return nl.back
}

// Len returns the number of nodes in the linked list.
func (nl *NodeList) Len() int {
	return nl.length
}

/* Insert pushes the given node to the back of the list. If the list is empty,
the given node becomes the head of the list.*/
func (nl *NodeList) Insert(n *Node) {
	if nl.front == nil {
		nl.front = n
		nl.back = nl.front
		nl.length = 1
		return
	}

	n.prev = nl.back
	n.Next = nil
	nl.back.Next = n
	nl.back = n

	nl.length++
}

/* Merge combines two node lists together. If the called node list is empty,
it becomes the other list. If the other list is empty, nothing changes.
Otherwise, the nodes of the other list are appended to this list. The other
 list is not changed. Note that this is an _unsorted_ merge. */
func (nl *NodeList) Merge(other *NodeList) {
	if other == nil || other.Front() == nil || other.length == 0 {
		return
	}

	if nl.front == nil {
		nl.front = other.Front()
		nl.back = other.Back()
		nl.length = other.length
		return
	}
	if other.length == 1 {
		nl.Insert(other.Front())
		return
	}
	nl.back.Next = other.Front()
	nl.back.Next.prev = nl.back
	nl.back = other.Back()

	nl.length += other.length
}

/*Remove takes the given node out of the list and returns it. */
func (nl *NodeList) Remove(n *Node) *Node {
	if n == nil || nl.length == 0 {
		return nil
	}

	if n.prev == nil {
		if nl.front == n {
			nl.front = n.Next
		}
	} else {
		n.prev.Next = n.Next
	}

	if n.Next == nil {
		if nl.back == n {
			nl.back = n.prev
		}
	} else {
		n.Next.prev = n.prev
	}

	n.Next = nil
	n.prev = nil

	nl.length--
	return n
}

/* String generates a string representation of the list. If the list is empty,
 String returns "(Empty list)". Otherwise, it recursively generates
"(Value[: children][, Value...])".*/
func (nl *NodeList) String() string {
	if nl.front == nil {
		return "(Empty list)"
	}

	s := ""
	for n := nl.Front(); n != nil; n = n.Next {
		s += n.String()
		if n.Next != nil {
			s += ", "
		}
	}
	return s
}
