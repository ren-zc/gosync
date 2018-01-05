package gosync

import (
	"flag"
	// "fmt"
	"bufio"
	"encoding/gob"
	"fmt"
	"log"
	"net"
)

func Client() {
	var updateMode bool
	var overwrite bool
	var deletion bool
	var targets string
	var lsnHost string
	var lsnPort string
	flag.StringVar(&lsnHost, "h", "127.0.0.1", "Please tell me the host ip which you want listen on.")
	flag.StringVar(&lsnPort, "p", "8999", "Please tell me the port which you want listen on.")
	flag.BoolVar(&updateMode, "u", false, "Please specify the sync mode. true/false.")
	flag.BoolVar(&overwrite, "o", false, "Whether the modified files will be overwriten.")
	flag.BoolVar(&deletion, "d", false, "Whether the redundant files will be deleted.")
	flag.StringVar(&targets, "t", "", "Please specify the target hosts.")
	flag.Parse()
	var userTask = Message{}
	userTask.MgID = RandId()
	userTask.MgType = "test"
	if updateMode {
		userTask.MgName = "UpdateSync"
		if overwrite {
			userTask.StrOption = "overwrite"
		}
		if deletion {
			userTask.StrOption = userTask.StrOption + "," + "deletion"
		}
	} else {
		userTask.MgName = "DefaultSync"
	}
	userTask.MgString = targets
	conn, cnErr := net.Dial("tcp", lsnHost+":"+lsnPort)
	if cnErr != nil {
		log.Fatalln(cnErr)
	}
	ghandleConn(conn, userTask)
}

func ghandleConn(conn net.Conn, mg Message) {
	defer conn.Close()
	cnRd := bufio.NewReader(conn)
	cnWt := bufio.NewWriter(conn)
	dec := gob.NewDecoder(cnRd)
	enc := gob.NewEncoder(cnWt)
	encErr := enc.Encode(mg)
	if encErr != nil {
		log.Println(encErr)
	}
	cnWt.Flush()

	// ...waiting server response...

	var newmg Message
	rcvErr := dec.Decode(&newmg)
	if rcvErr != nil {
		log.Println(rcvErr)
	}
	fmt.Println(newmg)
}
