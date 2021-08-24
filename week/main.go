package main

import (
	"fmt"
	"time"
)

func main() {
	t := time.Now()
	fmt.Println(int(t.Weekday()))
	fmt.Println(time.Now().Hour())
}
