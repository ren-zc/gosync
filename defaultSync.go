package gosync

import (
	"archive/zip"
	// "fmt"
	"bufio"
	"encoding/gob"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	// "sync"
)

type ret struct {
	status bool
	err    error
}

// md5 string
type md5s string

// type fileList []string
// type ips []string
type hostIP string

// type cnRet struct {
type hostRet struct {
	hostIP
	ret
}

// var md5OfFileList = map[md5s]fileList{}
type zipFileInfo struct {
	name string
	md5s
}
type transUnit struct { // 传输单元, 代表一组相同同步任务的主机列表
	hosts       []hostIP
	fileMd5List []string
	zipFileInfo
}
type diffInfo struct {
	md5s // diff文件列表的md5
	hostIP
	files []string // 需要更新的文件, 即diff文件列表
}

// var wg sync.WaitGroup
var allConn = map[hostIP]ret{}   // 用于收集host返回的sync结果
var retReady = make(chan string) // 从此channel读取到Done表示所有host已返回结果
var lg *log.Logger

func init() {
	lg = log.New(os.Stdout, "Err ", log.Lshortfile)
}

// 启用conn监控goroutine
var retCh = make(chan hostRet)

// 负责管理allConn
func cnMonitor(i int) {
	var c hostRet
	var l int
	for {
		c = <-retCh
		allConn[c.hostIP] = c.ret
		l = len(allConn)
		if l == i {
			// 相等表示所有host已返回sync结果
			break
		}
	}
	close(retCh)
	retReady <- "Done"
}

func putRetCh(host hostIP, err error) {
	var re ret
	if err != nil {
		re = ret{false, err}
	} else {
		re = ret{true, nil}
	}
	retCh <- hostRet{host, re}
}

func TravHosts(hosts []string, fileMd5List []string, flMd5 md5s, defaultSync bool) {
	hostNum := len(hosts)
	go cnMonitor(hostNum)

	// 和所有host建立连接
	var conn net.Conn
	var cnErr error
	var port = "8999"
	for _, host := range hosts {
		conn, cnErr = net.Dial("tcp", host+port)
		if cnErr != nil {
			// re := ret{false, cnErr}
			// retCh <- hostRet{hostIP(host), re}
			putRetCh(hostIP(host), cnErr)
			continue
		}
		// handle conn
		go hdRetConn(conn, fileMd5List, flMd5, defaultSync)
	}
}

var diffCh = make(chan diffInfo)

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
	if err != nil {
		lg.Printf("%s\t%s\n", conn.RemoteAddr().String(), err)
		putRetCh(hostIP(conn.RemoteAddr().String()), err)
	}
	err = cnWt.Flush()
	if err != nil {
		lg.Printf("%s\t%s\n", conn.RemoteAddr().String(), err)
		putRetCh(hostIP(conn.RemoteAddr().String()), err)
	}

	var hostMg Message

	for {

		err = dec.Decode(&hostMg)
		if err != nil {
			lg.Println(err)
			continue
		}

		if !defaultSync {
			// 接收file md5 list diff
			// 通过channel输出diff结果, diffCh
		}
		// 等待接收host的sync结果并通过channel发送到allConn的monitor
		// retCh
	}
}

// 返回文件md5列表
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
		if !info.IsDir() {
			md5Str, fErr = Md5OfAFile(path)
			if fErr != nil {
				lg.Println(fErr)
				return fErr
			}
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

func DefaultSync(mg *Message, hosts []string) (map[md5s]transUnit, error) {
	// 准备src文件列表, 如果mg中zip选项为true, 则同时返回zip文件md5 map
	tus := make(map[md5s]transUnit)
	var fileMd5List []string
	var zipFI zipFileInfo
	var traErr error
	fileMd5List, traErr = Traverse(mg.SrcPath)
	if traErr != nil {
		return nil, traErr
	}
	listMd5 := Md5OfASlice(md5s(fileMd5List))
	TravHosts(hosts, fileMd5List, listMd5, true)
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
	tus[listMd5] = tu
	// 返回tus
	return tus, nil
}

func UpdateSync(mg *Message, hosts []string) (map[md5s]transUnit, error) {
	var tus = make(map[md5s]transUnit)
	var fileMd5List []string
	var zipFI zipFileInfo
	var traErr error
	fileMd5List, traErr = Traverse(mg.SrcPath)
	if traErr != nil {
		return nil, traErr
	}
	listMd5 := Md5OfASlice(md5s(fileMd5List))
	TravHosts(hosts, fileMd5List, listMd5, true)
	di := diffInfo{}
	hostNum := len(hosts)
	for i := 0; i < hostNum; i++ {
		di = <-diffCh
		if len(di.files) == 0 {
			// re := ret{true, nil}
			// retCh <- hostRet{hostIP, re}
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
	// 返回tus
	return tus, nil
}
