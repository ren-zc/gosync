package main

import (
	"github.com/jacenr/gosync"
)

func main() {
	gosync.DebugFlag = true
	gosync.DubugInfor("test1", "\t", "test2")
}
