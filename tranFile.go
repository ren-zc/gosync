package gosync

import (
	// "os"
	"net"
)

var worker int

func tranFile(m md5s, tu *transUnit) {
	// 整理hostIP列表, 同时转换成 []string

	// 调用 tranFileTree(), 得到[]chan Message

	// 如果zip为true, 则开始传输zip文件, 包括zip的md5
	// tranPerFile(zip)

	// 如果zip为false, 对tu中的fileMd5List依次进行传输
	// for tranPerFile(file)

	// 待所有文件传输完毕, 关闭各个channel
}

func tranFileTree(hosts []string) []chan Message {
	// 从列表中取出worker个host进行连接

	// fileStreamChList := make([]chan Message, worker)
	// for ... make chan Message ... append ...
	// go hdTreeNode(conn, fileStreamChList[i])

	// 如果列表不为空, 把列表剩余host分成worker份通过channel分发出去
	// MgType: file, MgName: hostList, MgStrings: []string
	// --> msgFunc.go hdFile()
}

func hdTreeNode(conn net.Conn, fileStreamCh chan Message) {
	// defer close conn
	// gbc
	// 接收channel中的内容, 并进行分发
	// 如果收到重发请求...
	// 如果channel关闭, 关闭conn, 则退出goroutine
}

func tranPerFile(fileStreamChList []chan Message, fileName string, zipMd5 md5s) {
	// 打开文件
	// 把文件内容切片到Message, 依次分发到各个channel
}
