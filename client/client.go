package client

import (
	"bufio"
	"encoding/gob"
	"flag"
	"fmt"
	// "io/ioutil"
	"log"
	"net"
	"os"
)

type Message struct {
	MgID      int
	MgType    string // cmd,auth,file,info
	MgName    string // cmd name, auth username, file name
	MgContent string // cmd option, autho user passwd, file piece
	IntOption int    // file piece number or other
	StrOption string // start, continue, end
}

func Start() {
	var lsnHost string
	var lsnPort string
	flag.StringVar(&lsnHost, "h", "127.0.0.1", "Please tell me the host ip which you want listen on.")
	flag.StringVar(&lsnPort, "p", "9001", "Please tell me the port which you want listen on.")
	flag.Parse()
	conn, Err := net.Dial("tcp", lsnHost+":"+lsnPort)
	if Err != nil {
		log.Fatalln(Err)
	}
	defer conn.Close()
	cnRd := bufio.NewReader(conn)
	cnWt := bufio.NewWriter(conn)
	// cnBuf := bufio.NewReadWriter(conn, conn)
	enc := gob.NewEncoder(cnWt)
	dec := gob.NewDecoder(cnRd)
	m := Message{
		MgID:      1,
		MgType:    "auth",
		MgContent: "123456",
	}
	fmt.Println(m)
	encErr := enc.Encode(m)
	cnWt.Flush()
	if encErr != nil {
		fmt.Println(encErr)
	}
	p := []byte{}
	var rcvErr error
	f, fcErr := os.Create("tmp")
	defer f.Close()
	if fcErr != nil {
		log.Fatalln(fcErr)
	}
	fb := bufio.NewWriter(f)
	// var wtErr error
	for rcvErr == nil {
		rcvErr = dec.Decode(&p)
		// fmt.Println(p)
		fmt.Printf("%s\n", string(p))
		n, wtErr := fb.Write(p)
		fmt.Println(n)
		if wtErr != nil {
			fmt.Println(wtErr)
		}
	}
	fb.Flush()
}
