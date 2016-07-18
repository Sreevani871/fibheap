package node

import "testing"
import S "github.com/Sreevani871/fibheap/node"
import "fmt"

func TestInsert_Remove_AddChild_DeleteChild(t *testing.T) {
	nl := S.NewNodeList()

	n := S.NewNode(10.0, nil)
	nl.Insert(n)
	res := nl.Remove(n)
	if res != n {
		t.Error("Something went wrong")
	}

	nl.Insert(S.NewNode(20.0, nil))
	n2 := S.NewNode(30.0, nil)
	nl.Insert(n2)
	fmt.Println(nl)
	nl.Remove(n)
	fmt.Println(nl)
	n2.AddChild(n)
	if n.Parent != n2 {
		t.Error("Something went wrong")
	}
	fmt.Println(nl)

	n3 := S.NewNode(40.0, nil)
	nl.Insert(n3)
	nr := nl.Remove(S.NewNode(20.0, nil))
	fmt.Println(nl)
	n3.AddChild(nr)
	if nr.Parent != n3 {
		t.Error("Something went wrong")
	}
	n3.DeleteChild(nr)
	if nr.Parent != nil {
		t.Error("Something went wrong")
	}
	fmt.Println(nl)
}
