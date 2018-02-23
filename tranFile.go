package gosync

import (
	"bufio"
	"io"
	"net"
	"os"
	// "time"
)

var worker int

func tranFile(m md5s, tu *transUnit) {
	DubugInfor("in tranFile")

	// 整理hostIP列表, 同时转换成 []string
	// 调用 tranFileTree(), 得到[]chan Message
	hosts := make([]string, 0)
	for _, v := range tu.hosts {
		hosts = append(hosts, string(v))
	}
	treeChiledNode, ConnErrHost := tranFileTree(hosts)
	for _, host := range ConnErrHost {
		PrintInfor(host)
	}

	// 开始传送文件, 定义一个n, 用于累加所有文件的切片数量
	// 若是zip元文件, message中应附带mg.Zip字段
	// 若传输的文件本身就是一个zip文件, 切勿设置mg.Zip字段
	fileNames := []string{}
	Zip := false
	mgID := RandId()
	if tu.zipFileInfo.name != "" {
		Zip = true
		fileNames = append(fileNames, tu.zipFileInfo.name)
	} else {
		fileNames = tu.fileMd5List
	}

	n, _ := tranByFileList(treeChiledNode, fileNames, Zip, mgID)

	// 测试用的message
	// var mg0 Message
	// mg0.MgID = RandId()
	// mg0.MgType = "fileStream"
	// mg0.MgName = "testName"
	// mg0.MgString = string(m)
	// mg0.IntOption = 1
	// mg0.B = true
	// for _, ch := range treeChiledNode {
	// 	ch <- mg0
	// }

	// ******
	// 传送完成发送mg.MgString == "allEnd"的mg, 同时关闭fileStreamCh
	// IntOption标识传输的文件切片数量, 不包括IntOption本身所在的message
	// ******
	var mg Message
	mg.MgID = mgID
	mg.MgType = "fileStream"
	mg.IntOption = n
	mg.MgString = "allEnd"
	for _, ch := range treeChiledNode {
		ch <- mg
		close(ch)
	}
}

func tranFileTree(hosts []string) ([]chan Message, []string) {
	DubugInfor("in tranFileTree")
	// 从列表中取出worker个host进行连接
	fileSteamChList := make([]chan Message, 0)
	treeConnFailedList := make([]chan Message, 0)
	ConnErrHost := make([]string, 0)
	hostsLen := len(hosts)
	if hostsLen == 0 {
		return nil, ConnErrHost
	}
	var getHosts []string
	if hostsLen <= worker {
		getHosts = hosts
		hosts = make([]string, 0)
	} else {
		getHosts = hosts[:worker]
		hosts = hosts[worker:]
	}
	for _, h := range getHosts {
		conn, err := net.Dial("tcp", h)
		if err != nil {
			ConnErrHost = append(ConnErrHost, h)
			continue
		}
		fileStreamCh := make(chan Message)
		treeConnFailed := make(chan Message)
		fileSteamChList = append(fileSteamChList, fileStreamCh)
		treeConnFailedList = append(treeConnFailedList, treeConnFailed)
		go hdTreeNode(conn, fileStreamCh, treeConnFailed)
	}
	ChL := len(fileSteamChList)
	HoL := len(hosts)
	var mg Message
	mg.MgType = "hostList"
	if HoL != 0 {
		// 分发host list
		d := HoL / ChL
		m := HoL % ChL
		if m > 0 {
			d++
		}
		var n int
		for i := 0; i < ChL; i++ {
			if n == 1 {
				fileSteamChList[i] <- mg
				continue
			}
			limit := (i + 1) * d
			if limit >= HoL {
				limit = HoL
				n = 1
			}
			subHosts := hosts[i*d : limit]
			// 把subHosts封装到Message, 分发出去
			mg.MgStrings = subHosts
			fileSteamChList[i] <- mg
			DubugInfor("hostList mg send")
		}
	} else {
		for i := 0; i < ChL; i++ {
			fileSteamChList[i] <- mg
			DubugInfor("hostList mg send")
		}
	}
	// 接收下级主机的反馈
	for _, ch := range treeConnFailedList {
		mg := <-ch
		DubugInfor("get mg from treeConnFailed channel")
		if len(mg.MgStrings) != 0 {
			ConnErrHost = append(ConnErrHost, mg.MgStrings...)
		}
	}
	return fileSteamChList, ConnErrHost
}

func hdTreeNode(conn net.Conn, fileStreamCh chan Message, treeConnFailed chan Message) {
	DubugInfor("in hdTreeNode")
	defer conn.Close()
	gbc := initGobConn(conn)
	// 接收host list, 并分发到conn的另一端
	for {
		listMg, ok := <-fileStreamCh
		if !ok {
			// break // 主动break会close conn, 会不会导致下游node出现问题? 会!!!
			continue // contine, 直到下游node主动关闭conn
		}
		// 从fileStreamCh中接收mg, 分发到conn的另一端
		err := gbc.gobConnWt(listMg)
		if err != nil {
			// *** 待处理 ***
			PrintInfor(err)
		}
		if listMg.MgType == "hostList" {
			for {
				var connMg Message
				err := gbc.dec.Decode(&connMg)
				if err != nil {
					// *** 待处理 ***
					PrintInfor(err)
				}
				if connMg.MgString != "connRet" {
					continue
				}
				treeConnFailed <- connMg
				close(treeConnFailed)
				DubugInfor("get child node return")
				break
			}
		}
	}
	// 如果channel被关闭, 关闭conn, 退出goroutine
	DubugInfor("hdTreeNode closed.")
}

func tranByFileList(fileStreamChList []chan Message, fileNames []string, Zip bool, mgID int) (int, error) {
	var m int
	var err error
	var p = make([]byte, 4096)
	var e = make([]byte, 0)
	// var mgID = RandId()
	for _, file := range fileNames {
		var f *os.File
		var i int
		f, err = os.Open(file)
		if err != nil {
			// 待补充
		}
		r := bufio.NewReader(f)
		for {
			var fp Message
			var n int
			m++
			i++
			n, err = r.Read(p)
			if n > 0 {
				if n < 4096 {
					e = p[:n]
				} else {
					e = p
				}
			}
			if n == 0 { // 有可能是个空文件
				e = e[:0]
				fp.B = true
			}
			fp.MgID = mgID
			fp.MgType = "fileStream"
			fp.MgName = file
			fp.MgByte = e
			fp.IntOption = i
			fp.Zip = Zip
			for _, ch := range fileStreamChList {
				ch <- fp
				DubugInfor("send ", i)
			}
			if err == io.EOF {
				break
			} else {
				// 待补充错误处理代码
			}
		}
		f.Close()
	}
	return m, nil
}
