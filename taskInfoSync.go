package gosync

import (
	"encoding/gob"
	"fmt"
	"net"
)

// md5 string
type md5s string

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

// 将host的sync结果push到channel
func putRetCh(host hostIP, err error, retCh chan hostRet) {
	var re ret
	if err != nil {
		re = ret{false, err}
	} else {
		re = ret{true, nil}
	}
	retCh <- hostRet{host, re}
}

// 启动监控进程, 和各目标host建立连接
func TravHosts(hosts []string, fileMd5List []string, flMd5 md5s, mg *Message, diffCh chan diffInfo, retCh chan hostRet, taskID string) {

	var port = ":8999"
	for _, host := range hosts {
		conn, cnErr := net.Dial("tcp", host+port)
		// 建立连接失败, 即此目标host同步失败
		if cnErr != nil {
			putRetCh(hostIP(host), cnErr, retCh)
			continue
		}
		go hdRetConn(conn, fileMd5List, flMd5, mg, diffCh, retCh, taskID)
	}
}

// 发送源host的文件列表, 接收目标host的请求列表, 接收目标host的sync结果
// flMd5: md5 of fileMd5List
func hdRetConn(conn net.Conn, fileMd5List []string, flMd5 md5s, mg *Message, diffCh chan diffInfo, retCh chan hostRet, taskID string) {
	defer conn.Close()
	// 包装conn
	gbc := initGobConn(conn)

	// 发送fileMd5List
	var fileMd5ListMg Message
	fileMd5ListMg.TaskID = taskID
	fileMd5ListMg.MgID = RandId()
	fileMd5ListMg.MgType = "allFilesMd5List"
	fileMd5ListMg.MgString = string(flMd5)
	fileMd5ListMg.MgStrings = fileMd5List
	fileMd5ListMg.DstPath = mg.DstPath
	fileMd5ListMg.Del = mg.Del
	fileMd5ListMg.Overwrt = mg.Overwrt
	err := gbc.gobConnWt(fileMd5ListMg)
	// 如果encode失败, 则此conn对应的目标host同步失败
	if err != nil {
		lg.Printf("%s\t%s\n", conn.RemoteAddr().String(), err)
		putRetCh(hostIP(conn.RemoteAddr().String()), err, retCh)
		return
	}

	// 设置超时器, 1min
	fresher := make(chan struct{})
	ender := make(chan struct{})
	stop := make(chan struct{})
	go setTimer(fresher, ender, stop, 1)

	// 用于接收目标host发来的信息
	var hostMg Message
	dataRecCh := make(chan Message)
	go dataReciver(gbc.dec, dataRecCh)

	var diffFile diffInfo
	var diffFlag int
ENDCONN:
	for {
		select {
		case <-stop:
			// 超时失败
			err = fmt.Errorf("%s", "timeout 1")
			putRetCh(hostIP(conn.RemoteAddr().String()), err, retCh)
			if diffFlag != 1 {
				diffFile.files = nil
				diffCh <- diffFile
			}
			break ENDCONN
		case hostMg = <-dataRecCh:
			switch hostMg.MgType {
			case "result":
				if hostMg.b {
					err = nil
				} else {
					err = fmt.Errorf("%s", hostMg.MgString)
				}
				putRetCh(hostIP(conn.RemoteAddr().String()), err, retCh)
				ender <- struct{}{}
				break ENDCONN
			case "diffOfFilesMd5List":
				diffFile.files = hostMg.MgStrings
				diffFile.hostIP = hostIP(conn.RemoteAddr().String())
				diffFile.md5s = md5s(hostMg.MgString)
				diffCh <- diffFile
				diffFlag = 1
				fresher <- struct{}{}
			case "live": // heartbeat
				fresher <- struct{}{}
			}
		}
	}
}

func dataReciver(dec *gob.Decoder, dataRecCh chan Message) {
	var hostMessage Message
	for {
		err := dec.Decode(&hostMessage)
		if err != nil {
			// lg.Println(err)
			break
		}
		dataRecCh <- hostMessage
	}
}
