package gosync

import (
	"fmt"
	"runtime"
)

var DebugFlag bool

func DubugInfor(a ...interface{}) {
	if DebugFlag {
		var file string
		var line int
		var ok bool
		_, file, line, ok = runtime.Caller(1)
		if !ok {
			file = "???"
			line = 0
		}
		fmt.Printf("* %s:%d: ", file, line)
		for _, v := range a {
			fmt.Printf("%v", v)
		}
		fmt.Println()
	}
}
