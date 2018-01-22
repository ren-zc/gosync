package gosync

import (
	"bufio"
	"encoding/gob"
	"flag"
	"log"
	"net"
	"os"
)

var lg *log.Logger // 使log记录行号, 用于debug
var pwd string

func init() {
	lg = log.New(os.Stdout, "* ", log.Lshortfile)
	var err error
	pwd, err = os.Getwd()
	if err != nil {
		lg.Println(err)
	}
	q := []string{}
	t = &Tasks{q, "", Complated}
}

var worker int

type gobConn struct {
	cnRd *bufio.Reader
	cnWt *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}

func initGobConn(conn net.Conn) *gobConn {
	gbc := new(gobConn)
	gbc.cnRd = bufio.NewReader(conn)
	gbc.cnWt = bufio.NewWriter(conn)
	gbc.dec = gob.NewDecoder(gbc.cnRd)
	gbc.enc = gob.NewEncoder(gbc.cnWt)
	return gbc
}

func (gbc *gobConn) gobConnWt(mg Message) error {
	err := gbc.enc.Encode(mg)
	if err != nil {
		return err
	}
	err = gbc.cnWt.Flush()
	return err
}

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
	defer conn.Close()
	gbc := initGobConn(conn)
	var mg Message
	rcvErr := gbc.dec.Decode(&mg)
	if rcvErr != nil {
		lg.Println(rcvErr)
	}
	// fmt.Println(mg)

	// **deal with the mg**
	switch mg.MgType {
	case "task":
		hdTask(&mg, gbc, conn)
	case "file":
		hdFile(&mg, gbc)
	case "allFilesMd5List":
		hdFileMd5List(&mg, gbc)
	default:
		hdNoType(&mg, gbc)
	}
}
