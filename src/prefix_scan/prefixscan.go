package main

import "fmt"

func main() {

	c := []int{5, 3, -6, 2, 7, 10, -2, 8}
	fmt.Printf("your array is: %v", c)
	l := len(c)

	//a[i + 2^(d+1) - 1] := a[i + 2^d - 1] + a[i + 2^(d+1) - 1]
	// d=0
	for i, _ := range c {
		if 2*i < l {
			c[2*i+1] = c[2*i] + c[2*i+1]
		}
	}

	//d=1
	for i, _ := range c {
		if 4*i < l {
			c[4*i+3] = c[4*i+1] + c[4*i+3]
		}
	}

	//d=2
	for i, _ := range c {
		if 8*i < l {
			c[8*i+7] = c[8*i+3] + c[8*i+7]
		}
	}

	/*for i = 0 to n – 1 by 2d+1 do in parallel
	  temp := a[i + 2d - 1]
	  a[i + 2d - 1] := a[i + 2d+1 - 1] (left child)
	  a[i + 2d+1 - 1] := temp + a[i + 2d+1 - 1] (right child)
	*/

	c[7] = 0
	//d=2

	for i, _ := range c {
		if 8*i < l {
			temp := c[8*i+3]
			c[8*i+3] = c[8*i+7]        //(left child)
			c[8*i+7] = temp + c[8*i+7] //(right child)
		}
	}

	//d=1
	for i, _ := range c {
		if 4*i < l {
			temp := c[4*i+1]
			c[4*i+1] = c[4*i+3]        //(left child)
			c[4*i+3] = temp + c[4*i+3] //(right child)
		}
	}

	//d=0
	for i, _ := range c {
		if 2*i < l {
			temp := c[2*i]
			c[2*i] = c[2*i+1]          //(left child)
			c[2*i+1] = temp + c[2*i+1] //(right child)
		}
	}

	fmt.Printf("your prefix sum array: %v", c)
}
