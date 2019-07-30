package main

import (
	"fmt"
	"strings"
)

func main() {
	s := "${some_value} ${some_value_2}"
	fmt.Println(s)
	s = strings.TrimSpace(s)
	fmt.Println(s)
	s = strings.TrimLeft(s, "${")
	fmt.Println(s)
	s = strings.TrimRight(s, "}")
	fmt.Println(s)
	pos := strings.Index(s, ".")
	fmt.Println(pos)
}
