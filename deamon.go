package gosync

import (
	"bufio"
	"encoding/gob"
	"flag"
	"net"
	"os"
)

var cwd string
var transFilesAndMd5 map[string]string

// 接收本机最终同步结果, 并通过hdFileMd5List()发送给源主机, 全局变量, 不关闭
var hostRetCh chan Message

func init() {
	var err error
	cwd, err = os.Getwd()
	if err != nil {
		PrintInfor(err)
	}
	q := []string{}
	t = &Tasks{q, "", Complated}
	hostRetCh = make(chan Message)
	DebugFlag = true
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
		DubugInfor(err)
		os.Exit(1)
	}
	for {
		conn, err := svrln.Accept()
		if err != nil {
			DubugInfor(err)
			continue
		}
		go dhandleConn(conn)
	}
}

func dhandleConn(conn net.Conn) {
	defer conn.Close()
	gbc := initGobConn(conn)
	putCh := make(chan Message)
	var allPieces int
	var sendPieces int
CONNEND:
	for {
		var mg Message // ****** 若把mg写在for外部复用mg, 有BUG!!!, 两次写入的某些mg字段会被组合覆写!!! ******
		rcvErr := gbc.dec.Decode(&mg)
		if rcvErr != nil {
			DubugInfor(rcvErr)
		}
		switch mg.MgType {
		case "task":
			hdTask(&mg, gbc) // 发起任务
			break CONNEND
		case "hostList":
			getCh := make(chan Message)
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
			DubugInfor("fpbMonitor start")
			go hdFile(treeChiledNode, getCh)
		case "fileStream":
			if mg.MgString == "allEnd" {
				DubugInfor("allPieces setted")
				allPieces = mg.IntOption
			}
			DubugInfor(mg)
			putCh <- mg // ****** 若用channel传递指针有BUG!!!, 慎用 ******
			sendPieces++
			if allPieces > 0 && allPieces == (sendPieces-1) {
				close(putCh)
				DubugInfor("putCh closed")
				break CONNEND
			}
		case "allFilesMd5List":
			hdFileMd5List(&mg, gbc)
			DubugInfor(t)
			// *** 阻塞直到, 从channel读取同步结果 ***
			hR := <-hostRetCh
			DubugInfor("get hR")
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
		DubugInfor("hdFile get Message: ", mg)

		// 对所有message进行转发
		if treeChiledNode != nil {
			for _, ch := range treeChiledNode {
				ch <- mg
			}
		}

		// 仅对非"allEnd"的message进行保存
		if mg.MgString != "allEnd" {
			// *** 测试连接树 ***
			var hR Message
			hR.MgType = "result"
			hR.B = true
			hostRetCh <- hR
			DubugInfor("put hR")
			// ******************
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

	}
	DubugInfor("hdFile end")
}
