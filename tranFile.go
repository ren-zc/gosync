package gosync

import (
	// "os"
	"net"
)

var worker int

func tranFile(m md5s, tu *transUnit) {
	// 整理hostIP列表, 同时转换成 []string
	DubugInfor("in tranFile")
	ipList := make([]string, 0)
	for _, v := range tu.hosts {
		ipList = append(ipList, string(v))
	}

	// 调用 tranFileTree(), 得到[]chan Message
	hosts := make([]string, 0)
	for _, v := range tu.hosts {
		hosts = append(hosts, string(v))
	}
	treeChiledNode, ConnErrHost := tranFileTree(hosts)
	for _, host := range ConnErrHost {
		PrintInfor(host)
	}

	var mg0 Message
	mg0.MgID = RandId()
	mg0.MgType = "fileStream"
	mg0.MgName = "testName"
	mg0.MgString = string(m)
	mg0.IntOption = 1
	mg0.B = true
	for _, ch := range treeChiledNode {
		ch <- mg0
	}

	// 写入一条测试信息, 仅用于数据流网络的连接测试
	var mg Message
	mg.MgID = RandId()
	mg.MgType = "fileStream"
	mg.IntOption = 1
	mg.B = true
	mg.MgString = "allEnd"
	for _, ch := range treeChiledNode {
		ch <- mg
	}

	// ******
	// 传送完成发送mg.MgString == "allEnd"的mg, 同时关闭fileStreamCh
	// ******

	// for每一个文件, 打开, 读取, 遍历treeChiledNode分发

	// 如果zip为true, 则开始传输zip文件, 包括zip的md5
	// tranPerFile(zip)

	// 如果zip为false, 对tu中的fileMd5List依次进行传输
	// for tranPerFile(file)

	// 待所有文件传输完毕, 关闭各个channel
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
	// fileStreamChList := make([]chan Message, worker)
	// for ... make chan Message ... append ...
	// go hdTreeNode(conn, fileStreamChList[i])

	// 如果列表不为空, 把列表剩余host分成worker份通过channel分发出去
	// MgType: file, MgName: hostList, MgStrings: []string
}

func hdTreeNode(conn net.Conn, fileStreamCh chan Message, treeConnFailed chan Message) {
	DubugInfor("in hdTreeNode")
	defer conn.Close()
	gbc := initGobConn(conn)
	// 接收host list, 并分发到conn的另一端
	for {
		listMg, ok := <-fileStreamCh
		if !ok {
			close(treeConnFailed)
			break
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
				} else {
					treeConnFailed <- connMg
					close(treeConnFailed)
					DubugInfor("get child node return")
					break
				}
			}
		}
		// 将mg中的数据写入本地文件
	}
	// 接收conn另一端的连接反馈, 并通过channel传递到tranFileTree()
	// 接收文件数据流
	// 向conn另一端传递host list

	// 接收channel中的内容, 并进行分发
	// 如果收到重发请求...
	// 如果channel关闭, 关闭conn, 则退出goroutine
}

func tranPerFile(fileStreamChList []chan Message, fileName string, zipMd5 md5s) {
	// 打开文件
	// 把文件内容切片到Message, 依次分发到各个channel
}
