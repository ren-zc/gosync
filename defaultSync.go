package gosync

import (
	// "archive/zip"
	// "fmt"
	"bufio"
	"encoding/gob"
	"log"
	"net"
	"os"
	"path/filepath"
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

// zip文件名和md5
type zipFileInfo struct {
	name string
	md5s
}

// 传输单元, 代表一组相同同步任务的主机列表
type transUnit struct {
	hosts       []hostIP
	fileMd5List []string
	zipFileInfo
}

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
func TravHosts(hosts []string, fileMd5List []string, flMd5 md5s, defaultSync bool) {
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
		go hdRetConn(conn, fileMd5List, flMd5, defaultSync)
	}
}

var diffCh = make(chan diffInfo)

// 发送源host的文件列表, 接收目标host的请求列表, 接收目标host的sync结果
func hdRetConn(conn net.Conn, fileMd5List []string, flMd5 md5s, defaultSync bool) {
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
	if defaultSync {
		fileMd5ListMg.StrOption = "default"
	} else {
		fileMd5ListMg.StrOption = "update"
	}
	err := enc.Encode(fileMd5ListMg)
	// 如果encode失败, 则此conn对应的目标host同步失败
	if err != nil {
		lg.Printf("%s\t%s\n", conn.RemoteAddr().String(), err)
		putRetCh(hostIP(conn.RemoteAddr().String()), err)
	}
	err = cnWt.Flush()
	// 如果flush失败, 则此conn无法写入, 目标host同步失败
	if err != nil {
		lg.Printf("%s\t%s\n", conn.RemoteAddr().String(), err)
		putRetCh(hostIP(conn.RemoteAddr().String()), err)
	}

	// 用于接收目标host发来的信息
	var hostMg Message

	for {

		err = dec.Decode(&hostMg)
		if err != nil {
			lg.Println(err)
			continue
		}

		// if !defaultSync
		// 接收file md5 list diff
		// 通过channel输出diff结果, diffCh

		// 等待接收host的sync结果并通过channel发送到allConn的monitor
		// retCh
	}
}

// walk源host要同步的文件, 生成md5, 并返回列表
func Traverse(path string) ([]string, error) {
	f, fErr := os.Lstat(path)
	if fErr != nil {
		return nil, fErr
	}
	var dir string
	var base string
	if f.IsDir() {
		dir = path
		base = "."
	} else {
		dir = filepath.Dir(path)
		base = filepath.Base(path)
	}
	fErr = os.Chdir(dir)
	if fErr != nil {
		return nil, fErr
	}
	md5List := make([]string, 10)
	var md5Str string
	WalkFunc := func(path string, info os.FileInfo, err error) error {
		if info.Mode().IsRegular() {
			// if !info.IsDir() {
			md5Str, fErr = Md5OfAFile(path)
			if fErr != nil {
				lg.Println(fErr)
				return fErr
			}
			md5Str = path + "," + md5Str
			md5List = append(md5List, md5Str)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			md5Str = "symbolLink"
			md5Str = path + "," + md5Str
			md5List = append(md5List, md5Str)
		}
		return nil
	}
	fErr = filepath.Walk(base, WalkFunc)
	if fErr != nil {
		lg.Println(fErr)
		return nil, fErr
	}
	return md5List, nil
}

// 默认的同步模式, 不经过目标host的文件列表比对, 直接sync
func DefaultSync(mg *Message, targets []string) (map[md5s]transUnit, error) {
	// 准备src文件列表, 如果mg中zip选项为true, 则同时返回zip文件md5 map
	hosts := make([]hostIP, 1)
	for _, ipString := range targets {
		hosts = append(hosts, hostIP(ipString))
	}
	tus := make(map[md5s]transUnit)
	var fileMd5List []string
	var zipFI zipFileInfo
	var traErr error
	fileMd5List, traErr = Traverse(mg.SrcPath)
	if traErr != nil {
		return nil, traErr
	}
	listMd5 := Md5OfASlice(fileMd5List)
	TravHosts(targets, fileMd5List, md5s(listMd5), true)
	tu := transUnit{}
	tu.hosts = hosts
	tu.fileMd5List = fileMd5List
	if mg.Zip {
		zipName, zipMd5, traErr := Md5AndZip(fileMd5List)
		if traErr != nil {
			return nil, traErr
		}
		zipFI.name = zipName
		zipFI.md5s = zipMd5
	}
	tu.zipFileInfo = zipFI
	tus[md5s(listMd5)] = tu
	// 返回tus, 即传输任务单元, 默认同步模式, 所有目标host归属同一个传输任务单元
	return tus, nil
}

// 更新模式, 由目标host决定自己请求哪些文件, 源host正合请求列表, 返回传输任务map
func UpdateSync(mg *Message, targets []string) (map[md5s]transUnit, error) {
	hosts := make([]hostIP, 1)
	for _, ipString := range targets {
		hosts = append(hosts, hostIP(ipString))
	}
	var tus = make(map[md5s]transUnit)
	var fileMd5List []string
	var zipFI zipFileInfo
	var traErr error
	fileMd5List, traErr = Traverse(mg.SrcPath)
	if traErr != nil {
		return nil, traErr
	}
	listMd5 := Md5OfASlice(fileMd5List)
	TravHosts(targets, fileMd5List, md5s(listMd5), true)
	di := diffInfo{}
	hostNum := len(hosts)
	for i := 0; i < hostNum; i++ {
		di = <-diffCh
		if len(di.files) == 0 {
			putRetCh(di.hostIP, nil)
			continue
		}
		tu, ok := tus[di.md5s]
		if ok {
			tu.hosts = append(tu.hosts, di.hostIP)
		} else {
			h := []hostIP{}
			h = append(h, di.hostIP)
			tu.hosts = h
			tu.fileMd5List = di.files
			if mg.Zip {
				zipName, zipMd5, traErr := Md5AndZip(di.files)
				if traErr != nil {
					return nil, traErr
				}
				zipFI.name = zipName
				zipFI.md5s = zipMd5
			}
			tu.zipFileInfo = zipFI
			tus[di.md5s] = tu
		}
	}
	// 返回tus, 即一个传输任务单元
	return tus, nil
}
