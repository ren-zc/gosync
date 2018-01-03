package server

import (
	"bufio"
	"encoding/gob"
	"flag"
	"fmt"
	// "io/ioutil"
	"io"
	"log"
	"net"
	"os"
)

// type auth struct {
// 	name   string
// 	passwd string
// }

type Message struct {
	MgID      int
	MgType    string // cmd, auth, file, req, resp
	MgName    string // cmd name, auth username, file name
	MgContent string // cmd option, autho user passwd, file piece
	IntOption int    // file piece number or other
	StrOption string // start, continue, end
}

type userName string
type passwd string

var auth = map[userName]passwd{}

func Start() {
	var lsnHost string
	var lsnPort string
	flag.StringVar(&lsnHost, "h", "127.0.0.1", "Please tell me the host ip which you want listen on.")
	flag.StringVar(&lsnPort, "p", "9001", "Please tell me the port which you want listen on.")
	flag.Parse()
	svrln, err := net.Listen("tcp", lsnHost+":"+lsnPort)
	if err != nil {
		log.Fatalln(err)
	}
	for {
		conn, err := svrln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	cnRd := bufio.NewReader(conn)
	cnWt := bufio.NewWriter(conn)
	// netCn := bufio.NewReadWriter(conn, conn)
	dec := gob.NewDecoder(cnRd)
	enc := gob.NewEncoder(cnWt)
	var reciveData Message
	var rcvErr error
	// for {
	rcvErr = dec.Decode(&reciveData)
	if rcvErr != nil {
		fmt.Println(rcvErr)
	}
	fmt.Println(reciveData)
	f, fErr := os.Open("testfile/file3")
	if fErr != nil {
		log.Fatalln(fErr)
	}
	fmt.Println(f.Name())
	defer f.Close()
	fb := bufio.NewReader(f)
	p := make([]byte, 2048)
	var n int = 1
	var endErr error = fmt.Errorf("any")
	var sendErr error
	// fmt.Println(n != 0 && endErr != io.EOF)
	for n != 0 && endErr != io.EOF {
		n, endErr = fb.Read(p)
		// fmt.Println(n)
		// fmt.Println(p)
		if n < 2048 {
			// fmt.Println(endErr)
			p = p[:n]
		}
		fmt.Println(n)
		fmt.Printf("%s\n", string(p))
		sendErr = enc.Encode(p)
		if sendErr != nil {
			log.Println(sendErr)
		}
		cnWt.Flush()
	}
	// }
}
