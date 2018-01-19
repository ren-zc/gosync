package gosync

import (
	"flag"
	"fmt"
	"log"
	"net"
)

func Client() {
	var overwrite bool
	var deletion bool
	var zip bool
	var targets string
	var lsnHost string
	var lsnPort string
	var srcpath string
	var dstpath string
	flag.StringVar(&lsnHost, "h", "127.0.0.1", "Please tell me the host ip which you want listen on.")
	flag.StringVar(&lsnPort, "p", "8999", "Please tell me the port which you want listen on.")
	flag.StringVar(&srcpath, "src", ".", "Please specify the src file or directory path.")
	flag.BoolVar(&overwrite, "o", true, "Whether the modified files will be overwriten.")
	flag.BoolVar(&deletion, "d", false, "Whether the redundant files will be deleted.")
	flag.BoolVar(&zip, "z", false, "Whether the redundant files will be compressed.")
	flag.StringVar(&targets, "t", "", "Please specify the target hosts.")
	flag.StringVar(&dstpath, "dst", "/tmp", "Please specify the target host dst path.")
	flag.Parse()
	var userTask = Message{}
	userTask.MgID = RandId()
	userTask.MgType = "task"
	userTask.MgName = "sync"
	if overwrite {
		userTask.Overwrt = true
	}
	if deletion {
		userTask.Del = true
	}
	if zip {
		userTask.Zip = true
	}
	userTask.MgString = targets
	userTask.SrcPath = srcpath
	userTask.DstPath = dstpath
	conn, cnErr := net.Dial("tcp", lsnHost+":"+lsnPort)
	if cnErr != nil {
		log.Fatalln(cnErr)
	}
	ghandleConn(conn, userTask)
}

func ghandleConn(conn net.Conn, mg Message) {
	defer conn.Close()
	gbc := initGobConn(conn)
	encErr := gbc.gobConnWt(mg)
	if encErr != nil {
		log.Println(encErr)
	}

	// ...waiting server response...

	var newmg Message
	rcvErr := gbc.dec.Decode(&newmg)
	if rcvErr != nil {
		log.Println(rcvErr)
	}
	fmt.Println(newmg)
}
