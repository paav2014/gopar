package main

import "fmt"

func interpol(x []float64, y []float64) (new_x []float64, new_y []float64) {
	n := len(x)

	new_x = make([]float64, n*2-1)
	new_y = make([]float64, n*2-1)

	for i, _ := range x {
		new_x[2*i] = x[i]
		new_y[2*i] = y[i]
	}

	for i, _ := range x {
		if i >= n-1 {
			break
		}
		new_x[2*i+1] = (new_x[2*i] + new_x[2*i+2]) / 2
		new_y[2*i+1] = (new_y[2*i] + new_y[2*i+2]) / 2
	}
	return
}

func main() {
	x := []float64{0, 2, 6, 9}
	y := []float64{13, 2, 50, 4}

	fmt.Printf("X is: %+v\n", x)
	fmt.Printf("Y is: %+v\n", y)

	new_x, new_y := interpol(x, y)

	fmt.Printf("X' is: %+v\n", new_x)
	fmt.Printf("Y' is: %+v\n", new_y)
}
