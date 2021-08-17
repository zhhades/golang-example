package main

import "fmt"

func Add(a, b int) int {
	return a + b
}

type Company struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}
type Employee struct {
	Name    string  `json:"name"`
	company Company `json:"company"`
}

func main() {
	fmt.Println(Employee{
		Name: "zhh",
		company: Company{
			Name:    "mkw",
			Address: "beijing",
		},
	})
}
