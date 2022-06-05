package main

import "fmt"

func main() {
	var a *int
	fmt.Printf("%T\n", &a)
	fmt.Println(&a)
}
