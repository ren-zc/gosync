package gosync

import (
	// "os"
	"net"
)

var worker int

func tranFile(m md5s, tu *transUnit) {
	// 整理hostIP列表, 同时转换成 []string
	lg.Println("in tranFile")
	lg.Printf("worker %d\n", worker)
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
		lg.Println(host) // 先输入查看连接错误的主机
	}

	// 写入一条测试信息, 仅用于数据流网络的连接测试
	var mg Message
	mg.MgID = RandId()
	mg.MgType = "fileStream"
	mg.MgString = string(m)
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
	lg.Println("in tranFileTree")
	// 从列表中取出worker个host进行连接
	fileSteamChList := make([]chan Message, 0)
	treeConnFailedList := make([]chan Message, 0)
	ConnErrHost := make([]string, 0)
	hostsLen := len(hosts)
	var getHosts []string
	if hostsLen <= worker {
		getHosts = hosts
		hosts = make([]string, 0)
	} else {
		getHosts = hosts[:worker]
		hosts = hosts[worker:]
	}
	for _, h := range getHosts {
		lg.Println(h)
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
	if len(hosts) != 0 {
		lg.Println(len(hosts))
		// 分发host list
		ChL := len(fileSteamChList)
		HoL := len(hosts)
		d := HoL / ChL
		m := HoL % ChL
		if m > 0 {
			d++
		}
		for i := 0; i < ChL; i++ {
			limit := (i + 1) * d
			if limit >= HoL {
				limit = HoL
			}
			subHosts := hosts[i*d : limit]
			// 把subHosts封装到Message, 分发出去
			var mg Message
			mg.MgType = "hostList"
			mg.MgStrings = subHosts
			fileSteamChList[i] <- mg
			lg.Println("hostList mg send")
			if limit == HoL {
				break
			}
		}
	}
	// 接收下级主机的反馈
	for _, ch := range treeConnFailedList {
		mg := <-ch
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
	lg.Println("in hdTreeNode")
	defer conn.Close()
	gbc := initGobConn(conn)
	// 接收host list, 并分发到conn的另一端
	for {
		lg.Println("in hdTreeNode for")
		listMg, ok := <-fileStreamCh
		if !ok {
			break
		}
		lg.Println(listMg)
		// 从fileStreamCh中接收mg, 分发到conn的另一端
		err := gbc.gobConnWt(listMg)
		if err != nil {
			// *** 待处理 ***
			lg.Println(err)
		}
		if listMg.MgType == "hostList" {
			for {
				var connMg Message
				err := gbc.dec.Decode(&connMg)
				if err != nil {
					// *** 待处理 ***
					lg.Println(err)
				}
				if connMg.MgString != "connRet" {
					continue
				} else {
					treeConnFailed <- connMg
					close(treeConnFailed)
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
