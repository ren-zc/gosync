package main

import (
	"fmt"
	"github.com/jacenr/gosync"
)

func main() {
	// gosync.RandId()
	// fmt.Println(gosync.RandId())
	str, err := Zipfiles("/data/mygo/src/github.com/jacenr/filediff")
	fmt.Println(str)
	fmt.Println(err)
}
