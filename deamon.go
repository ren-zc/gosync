package gosync

import (
	"bufio"
	"encoding/gob"
	"flag"
	"log"
	"net"
	"os"
	// "sync"
)

var lg *log.Logger // 使log记录行号, 用于debug
var cwd string
var transFilesMd5 map[string]string

// 接收本机最终同步结果, 并通过hdFileMd5List()发送给源主机
var hostRetCh chan Message

func init() {
	lg = log.New(os.Stdout, "* ", log.Lshortfile)
	var err error
	cwd, err = os.Getwd()
	if err != nil {
		lg.Println(err)
	}
	q := []string{}
	t = &Tasks{q, "", Complated}
	hostRetCh = make(chan Message)
	// worker = 1
}

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

func (gbc *gobConn) gobConnWt(mg interface{}) error {
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
	// var treeChiledNode []chan Message
	// var fpb *filePieceBuf
	putCh := make(chan *Message)
CONNEND:
	for {
		switch mg.MgType {
		case "task":
			hdTask(&mg, gbc) // 发起任务
			break CONNEND
		case "hostList":
			getCh := make(chan *Message)
			fileTransEnd := make(chan struct{})
			hosts := mg.MgString
			treeChiledNode := tranFileTree(hosts)
			fpb := newFpb()
			go fpbMonitor(fpb, putCh, getCh)
			go hdFile(treeChiledNode, getCh)
			<-fileTransEnd
			close(putCh)
			break CONNEND
		case "fileStream":
			putCh <- &mg
			// hdFileStream(&mg, gbc)
			// break CONNEND
		case "allFilesMd5List":
			hdFileMd5List(&mg, gbc)
			break CONNEND
		default:
			hdNoType(&mg, gbc)
		}
	}
}

func hdFile(treeChiledNode []chan Message, getCh chan *Message, fileTransEnd chan struct{}) {
	var mg *Message
	for {
		mg = <-getCh
		if mg == nil {
			continue
		}
		// 分发和保存
		//
		// MgType: fileStream
		// 接收到Message, 将其中的文件内容保存到本地
		// 同时将Message原封不动的转发到[]chan Message的channel中

		// 如果是zip文件, 比对zip文件的md5, 比对后进行解压

		// 传输完成后, 和map transFilesMd5中的md5做对比
		// 如果md5不匹配, 则向gbc返回重新发送文件的请求
		// 重发请求格式:
		// *** 待定 ***
		// 暂定: log输出看结果分析原因

		// 所有完成后, 将结果通知到retCh chan hostRet

		// 退出
		if mg.MgString == "allEnd" {
			close(fileTransEnd)
			break
		}
	}
}
