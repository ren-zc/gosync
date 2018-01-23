package main

import (
	"fmt"
	"github.com/jacenr/gosync"
)

func main() {
	s := gosync.GetLocalIP()
	fmt.Println(s)
}
