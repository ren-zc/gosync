package gosync

import (
	"bufio"
	"encoding/gob"
	"flag"
	"log"
	"net"
	"os"
	// "time"
	// "sync"
)

var lg *log.Logger // 使log记录行号, 用于debug
var cwd string
var transFilesAndMd5 map[string]string

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
	flag.IntVar(&worker, "-n", 1, "The worker number.")
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
	// fmt.Println(mg)

	// **deal with the mg**
	// var treeChiledNode []chan Message
	// var fpb *filePieceBuf
	putCh := make(chan Message)
	var allPieces int
	var sendPieces int
CONNEND:
	for {
		var mg Message // ****** 把mg写在for外部复用mg, 有BUG!!!, 两次写入的某些mg字段会被组合覆写!!! ******
		rcvErr := gbc.dec.Decode(&mg)
		if rcvErr != nil {
			lg.Println(rcvErr)
		}
		switch mg.MgType {
		case "task":
			hdTask(&mg, gbc) // 发起任务
			break CONNEND
		case "hostList":
			lg.Println("in deamon hostList")
			getCh := make(chan Message)
			// fileTransEnd := make(chan struct{})
			hosts := mg.MgStrings
			treeChiledNode, ConnErrHost := tranFileTree(hosts)
			var connMg Message
			connMg.MgType = "hostList"
			connMg.MgStrings = ConnErrHost
			connMg.MgString = "connRet"
			err := gbc.gobConnWt(connMg)
			if err != nil {
				// *** 记录本地日志 ***
			}
			fpb := newFpb()
			go fpbMonitor(fpb, putCh, getCh)
			lg.Println("fpbMonitor start")
			go hdFile(treeChiledNode, getCh)
			// <-fileTransEnd
			// close(putCh)
			// break CONNEND
		case "fileStream":
			// 退出
			if mg.MgString == "allEnd" {
				// close(putCh)
				// break CONNEND
				lg.Println("allPieces setted")
				allPieces = mg.IntOption
			}
			lg.Println("get fileStream")
			lg.Println(mg)
			lg.Println(mg.MgStrings)
			putCh <- mg // ****** 若用channel传递指针有BUG!!!, 慎用 ******
			lg.Println("send fileStream")
			sendPieces++
			lg.Println(allPieces)
			lg.Println(sendPieces)
			lg.Println(allPieces > 0 && allPieces == (sendPieces-1))
			lg.Println(allPieces > 0 && allPieces == sendPieces)
			lg.Println(allPieces > 0 && allPieces == (sendPieces+1))
			if allPieces > 0 && allPieces == (sendPieces-1) {
				close(putCh)
				lg.Println("putCh closed")
				break CONNEND
			}
			// time.Sleep(10 * time.Second) // for debug
			// hdFileStream(&mg, gbc)
			// break CONNEND
		case "allFilesMd5List":
			hdFileMd5List(&mg, gbc)
			// *** 阻塞直到, 从channel读取同步结果 ***
			hR := <-hostRetCh
			lg.Println("get hR")
			err := gbc.gobConnWt(hR)
			if err != nil {
				// *** 记录本地日志 ***
			}
			break CONNEND
		default:
			hdNoType(&mg, gbc)
		}
	}
}

func hdFile(treeChiledNode []chan Message, getCh chan Message) {
	var mg Message
	var ok bool
	for {
		mg, ok = <-getCh
		if !ok {
			break
		}
		if mg.MgType != "fileStream" {
			continue
		}
		lg.Println(mg)

		// lg.Println("hd File get mg")
		if treeChiledNode != nil {
			for _, ch := range treeChiledNode {
				ch <- mg
			}
		}

		// *** 测试连接树 ***
		var hR Message
		hR.MgType = "result"
		hR.B = true
		hostRetCh <- hR
		lg.Println("put hR")
		// ******************

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

	}
	lg.Println("hdFile end")
}
