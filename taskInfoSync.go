package gosync

import (
	// "archive/zip"
	"bufio"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sort"
	// "strconv"
	// "sync"
)

// md5 string
type md5s string

// ip addr
type hostIP string

// host返回的结果
type ret struct {
	status bool
	err    error
}

// host返回的结果
type hostRet struct {
	hostIP
	ret
}

var allConn = map[hostIP]ret{}   // 用于收集host返回的sync结果
var retReady = make(chan string) // 从此channel读取到Done表示所有host已返回结果

// host返回的请求文件列表
type diffInfo struct {
	md5s // diff文件列表的md5
	hostIP
	files []string // 需要更新的文件, 即diff文件列表
}

var lg *log.Logger // 使log记录行号, 用于debug

func init() {
	lg = log.New(os.Stdout, "Err ", log.Lshortfile)
}

// 用于接收各host的sync结果
var retCh = make(chan hostRet)

// 负责管理allConn, 使用retCh channel
func cnMonitor(i int) {
	var c hostRet
	var l int
	for {
		c = <-retCh
		allConn[c.hostIP] = c.ret
		l = len(allConn)
		// 相等表示所有host已返回sync结果
		if l == i {
			break
		}
	}
	close(retCh)
	retReady <- "Done"
}

// 将host的sync结果push到channel
func putRetCh(host hostIP, err error) {
	var re ret
	if err != nil {
		re = ret{false, err}
	} else {
		re = ret{true, nil}
	}
	retCh <- hostRet{host, re}
}

// 启动监控进程, 和各目标host建立连接
func TravHosts(hosts []string, fileMd5List []string, flMd5 md5s, mg *Message) {
	hostNum := len(hosts)
	go cnMonitor(hostNum)

	// 和所有host建立连接
	var conn net.Conn
	var cnErr error
	var port = "8999"
	for _, host := range hosts {
		conn, cnErr = net.Dial("tcp", host+port)
		// 建立连接失败, 即此目标host同步失败
		if cnErr != nil {
			putRetCh(hostIP(host), cnErr)
			continue
		}
		go hdRetConn(conn, fileMd5List, flMd5, mg)
	}
}

var diffCh = make(chan diffInfo)

// 发送源host的文件列表, 接收目标host的请求列表, 接收目标host的sync结果
// flMd5: md5 of fileMd5List
func hdRetConn(conn net.Conn, fileMd5List []string, flMd5 md5s, mg *Message) {
	defer conn.Close()
	// 包装conn
	cnRd := bufio.NewReader(conn)
	cnWt := bufio.NewWriter(conn)
	dec := gob.NewDecoder(cnRd)
	enc := gob.NewEncoder(cnWt)

	// 发送fileMd5List
	var fileMd5ListMg Message
	fileMd5ListMg.MgID = RandId()
	fileMd5ListMg.MgType = "allFilesMd5List"
	fileMd5ListMg.MgString = string(flMd5)
	fileMd5ListMg.MgStrings = fileMd5List
	fileMd5ListMg.DstPath = mg.DstPath
	fileMd5ListMg.Del = mg.Del
	fileMd5ListMg.Overwrt = mg.Overwrt
	err := enc.Encode(fileMd5ListMg)
	// 如果encode失败, 则此conn对应的目标host同步失败
	if err != nil {
		lg.Printf("%s\t%s\n", conn.RemoteAddr().String(), err)
		putRetCh(hostIP(conn.RemoteAddr().String()), err)
		return
	}
	err = cnWt.Flush()
	// 如果flush失败, 则此conn无法写入, 目标host同步失败
	if err != nil {
		lg.Printf("%s\t%s\n", conn.RemoteAddr().String(), err)
		putRetCh(hostIP(conn.RemoteAddr().String()), err)
		return
	}

	// 设置超时器, 10min
	fresher := make(chan struct{})
	ender := make(chan struct{})
	stop := make(chan struct{})
	go setTimer(fresher, ender, stop, 10)

	// 用于接收目标host发来的信息
	var hostMg Message
	dataRecCh := make(chan Message)
	go dataReciver(dec, dataRecCh)

	var diffFile diffInfo
	var diffFlag int
END:
	for {
		select {
		case <-stop:
			// 超时失败
			err = fmt.Errorf("%s", "timeout 10")
			putRetCh(hostIP(conn.RemoteAddr().String()), err)
			if diffFlag == 1 {
				diffFile.files = nil
			}
			break END
		case hostMg <- dataRecCh:
			switch hostMg.MgType {
			case "result":
				if hostMg.b {
					err = nil
				} else {
					err = fmt.Errorf("%s", hostMg.MgString)
				}
				putRetCh(hostIP(conn.RemoteAddr().String()), err)
				ender <- struct{}{}
				break END
			case "diffOfFilesMd5List":
				diffFile.files = hostMg.MgStrings
				diffFile.hostIP = hostIP(conn.RemoteAddr().String())
				diffFile.md5s = hostMg.MgString
				diffCh <- diffFile
				diffFlag = 1
				fresher <- struct{}{}
			case "live": // backup use.
				fresher <- struct{}{}
			}
		}
	}
}

func dataReciver(dec gob.Decoder, dataRecCh chan Message) {
	var hostMessage Message
	for {
		err = dec.Decode(&hostMessage)
		if err != nil {
			lg.Println(err)
			break
		}
		dataRecCh <- hostMessage
	}
}
