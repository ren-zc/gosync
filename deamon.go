package gosync

import (
	"bufio"
	"encoding/gob"
	// "errors"
	"flag"
	"net"
	"os"
	"runtime"
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
	hostRetCh = make(chan Message, 1)
	DebugFlag = true
	goTus = 4
	// gob.Register(errors.New("")) // gob register errors.errorString bug
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
	defer runtime.GC()
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
			DubugInfor(t)
			// 返回任务开始前的目录
			err := os.Chdir(cwd)
			if err != nil {
				PrintInfor(err)
			}
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
			go fpbMonitor(fpb, putCh, getCh) // 对收到的文件切片进行缓冲排序
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
			break CONNEND
		default:
			hdNoType(&mg, gbc)
		}
	}
}

func hdFile(treeChiledNode []chan Message, getCh chan Message) {
	var mg Message
	var ok bool
	var f *os.File
	var err error
	var currentFile string
	var Zip bool
	for {
		mg, ok = <-getCh
		if !ok {
			// 从getCh读取失败, 说明所有文件读取完成
			if treeChiledNode != nil {
				for _, ch := range treeChiledNode {
					close(ch)
				}
			}
			break
		}
		// if mg.MgType != "fileStream" {
		// 	continue
		// }
		DubugInfor("hdFile get Message: ", mg)

		// 对所有message进行转发
		if treeChiledNode != nil {
			for _, ch := range treeChiledNode {
				ch <- mg
			}
		}

		// 仅对非"allEnd"的message进行保存
		if mg.MgString != "allEnd" {
			// 保存文件切片到本地
			if mg.IntOption == 1 {
				currentFile = mg.MgName
				Zip = mg.Zip
				f, err = os.Create(mg.MgName)
				if err != nil {
					// 处理错误
				}
			}
			_, err = f.Write(mg.MgByte)
			if err != nil {
				// 处理错误
			}
			if mg.B {
				err = f.Close()
				if err != nil {
					// 处理错误
				}
			}
		}
	}

	DubugInfor(os.Getwd()) // 查看当前目录是否是dst目录

	// 若zip选项为true, 解压缩
	if Zip {
		Unzip(currentFile)
	}

	// 对传输的文件进行校验
	// *** 校验代码写在此 ***
	// 校验过程: 对每个文件先生成md5, 然后和map transFilesMd5中的md5做对比
	// 若有一个文件的md5不匹配, 则打印不匹配的项, 待所有文件比对完成传输失败的结果
	for file, md5 := range transFilesAndMd5 {
		m, err := Md5OfAFile(file)
		if err != nil {
			// 待补充代码
			PrintInfor(file, " ", err)
		}
		if m == md5 {
			DubugInfor(file, " ", true)
		} else {
			DubugInfor(file, " ", false)
		}
	}

	// 若文件比对成功, 则发送成功的message(如下所示)到hostRetCh
	var hR Message
	hR.MgType = "result"
	hR.B = true
	hostRetCh <- hR
	DubugInfor("put hR")
	DubugInfor("hdFile end")
}
