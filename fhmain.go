package main

import S "github.com/Sreevani871/fibheap"
import "fmt"

func main() {
	node := S.NewHeap()
	fmt.Println("Choose Operation\n\n1)Insert\n2)ExtractMin\n3)GetMinimumValue\n4)Merge\n5)Print\n")
	var op int
	fmt.Scanf("%d", &op)
	for true {
		switch op {
		case 1:
			{
				fmt.Println("Enter an element to insert:")
				var e float64
				fmt.Scanf("%f", &e)
				node.Insert(e)
			}
		case 2:
			{
				fmt.Println(node.ExtractMin())
			}
		case 3:
			{
				fmt.Println(node.GetMinValue())
			}
		case 4:
			{
				node1 := S.NewHeap()
				node1.Insert(10.0)
				node1.Insert(20.0)
				node1.Insert(5)
				node.Merge(node1)
			}
		case 5:
			{
				node.String()
				node.PrintTrees()

			}
		default:
			{
			}
		}

		fmt.Println("choose operation")
		fmt.Scanf("%d", &op)
		if op > 5 {
			break
		}
	}
}
