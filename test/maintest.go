package main

import (
	"fmt"
	"github.com/jacenr/gosync"
)

func main() {
	// gosync.RandId()
	// fmt.Println(gosync.RandId())
	// str, err := gosync.Zipfiles("/data/mygo/src/github.com/jacenr/filediff/LICENSE")
	// fmt.Println(str)
	// fmt.Println(err)
	// s, _ := gosync.Md5OfAFile("gosync架构")
	// fmt.Println(s)
	s1, e1 := gosync.Traverse("/etc")
	// s1, e1 := gosync.Traverse("/data/mygo/src/github.com/jacenr/filediff")
	for _, v := range s1 {
		fmt.Println(v)
	}
	// fmt.Println(s1)
	// fmt.Println(m1)
	fmt.Println(e1)
}
