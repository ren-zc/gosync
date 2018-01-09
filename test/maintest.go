package main

import (
	"fmt"
	"github.com/jacenr/gosync"
)

func main() {
	// gosync.RandId()
	// fmt.Println(gosync.RandId())
	str, err := gosync.Zipfiles("/data/mygo/src/github.com/jacenr/filediff/LICENSE")
	fmt.Println(str)
	fmt.Println(err)
}
