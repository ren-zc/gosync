package gosync

import (
	"bufio"
	"encoding/gob"
	"flag"
	// "fmt"
	"log"
	"net"
	"os"
)

var lg *log.Logger // 使log记录行号, 用于debug

func init() {
	lg = log.New(os.Stdout, "*", log.Lshortfile)
}

var worker int

func DeamonStart() {
	var lsnHost string
	var lsnPort string
	flag.StringVar(&lsnHost, "h", "", "Please tell me the host ip which you want listen on.")
	flag.StringVar(&lsnPort, "p", "8999", "Please tell me the port which you want listen on.")
	flag.IntVar(&worker, "-n", 4, "The worker number.")
	flag.Parse()
	svrln, err := net.Listen("tcp", lsnHost+":"+lsnPort)
	if err != nil {
		lg.Fatalln(err)
	}
	for {
		conn, err := svrln.Accept()
		if err != nil {
			lg.Println(err)
			continue
		}
		go dhandleConn(conn)
	}
}

func dhandleConn(conn net.Conn) {
	lg.Println(conn.RemoteAddr().String()) // ****** test ******
	defer conn.Close()
	cnRd := bufio.NewReader(conn)
	cnWt := bufio.NewWriter(conn)
	dec := gob.NewDecoder(cnRd)
	enc := gob.NewEncoder(cnWt)
	var mg Message
	rcvErr := dec.Decode(&mg)
	if rcvErr != nil {
		lg.Println(rcvErr)
	}
	// fmt.Println(mg)

	// **deal with the mg**
	switch mg.MgType {
	case "task":
		hdTask(&mg, cnRd, cnWt, dec, enc)
	case "file":
		hdFile(&mg, cnRd, cnWt, dec, enc)
	case "allFilesMd5List":
		hdFileMd5List(&mg, cnRd, cnWt, dec, enc)
	default:
		hdNoType(&mg, cnRd, cnWt, dec, enc)
	}
}
