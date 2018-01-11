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
type fileList []string
type ips []string
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
type transUnit struct {
	hosts       []hostIP
	fileMd5List []string
	zipFileInfo
}
type diffInfo struct {
	md5s
	hostIP
	files []string
}

// var wg sync.WaitGroup
var allConn = map[hostIP]ret{}
var retReady = make(chan string) // 从此channel读取到Done表示所有host已返回结果
var lg *log.Logger

func init() {
	lg = log.New(os.Stdout, "Err ", log.Lshortfile)
}

// 负责管理allConn
func cnMonitor(ch chan hostRet, i int) {
	var c hostRet
	var l int
	for {
		c = <-ch
		allConn[c.hostIP] = c.ret
		l = len(allConn)
		if l == i {
			// 相等表示所有host已返回结果
			break
		}
	}
	close(ch)
	retReady <- "Done"
}

func TravHosts(hosts []string, mg *Message, defaultSync bool) (map[md5s]transUnit, error) {
	// 准备src文件列表, 如果mg中zip选项为true, 则同时返回zip文件md5 map
	tus := make(map[md5s]transUnit)
	var fileMd5List []string
	var zipFI zipFileInfo
	var traErr error
	if defaultSync {
		fileMd5List, zipFI, traErr = Traverse(mg.SrcPath, mg.Zip)
		listMd5 := Md5OfASlice(fileMd5List)
		tu := transUnit{}
		tu.hosts = hosts
		// tu.ListMd5 = listMd5
		tu.fileMd5List = fileMd5List
		tu.zipFileInfo = zipFI
		tus[listMd5] = tu
	} else {
		// updateSync模式, 第一次生成file md5 list仅用于各host比对, 无须zip文件
		fileMd5List, _, traErr = Traverse(mg.SrcPath, false)
	}
	if traErr != nil {
		return nil, traErr
	}

	// 启用conn监控goroutine
	retCh := make(chan hostRet)
	hostNum := len(hosts)
	go cnMonitor(retCh, hostNum)

	// 和所有host建立连接
	var conn net.Conn
	var cnErr error
	var port = "8999"
	var diffCh = make(chan diffInfo)
	for _, host := range hosts {
		conn, cnErr = net.Dial("tcp", host+port)
		if cnErr != nil {
			re := ret{false, cnErr}
			retCh <- hostRet{hostIP(host), re}
			continue
		}
		// handle conn
		go hdRetConn(retCh, diffCh, conn, fileMd5List, defaultSync)
	}

	// for循环接收每个goroutine的diff结果并整理, 前提是defaultSync为false
	// 即为更新模式
	di := diffInfo{}
	if !defaultSync {
		for i := 0; i < hostNum; i++ {
			di = <-diffCh
			tu, ok := tus[di.md5s]
			if ok {
				tu.hosts = append(tu.hosts, di.hostIP)
			} else {
				h := []hostIP{}
				h = append(h, di.hostIP)
				tu.hosts = h
				tu.fileMd5List = di.files
				tus[di.md5s] = tu
			}
		}
	}

	// 返回tus
	return tus, nil
}

func hdRetConn(ch chan hostRet, diffCh chan diffInfo, conn net.Conn, fileMd5List []string, defaultSync bool) {
	defer conn.Close()
	// 包装conn
	cnRd := bufio.NewReader(conn)
	cnWt := bufio.NewWriter(conn)
	dec := gob.NewDecoder(cnRd)
	enc := gob.NewEncoder(cnWt)

	// 发送fileMd5List, 作为文件发送
	// 首先发送一些元数据信息包括md5, 一些选项
	// 消息ID要一致

	if !defaultSync {
		// 接收file md5 list diff
		// 通过channel输出diff结果
	}
	// 等待接收host的sync结果并通过channel发送到allConn的monitor
}

// 返回src的md5文件列表, 如果有zip选项就同时返回以zip文件名为key, md5为value的map
// 出错则返回nil,nil,error
func Traverse(path string, zipOpt bool) ([]string, zipFileInfo, error) {
	f, fErr := os.Lstat(path)
	if fErr != nil {
		return nil, zipFileInfo{}, fErr
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
		return nil, zipFileInfo{}, fErr
	}
	md5List := make([]string, 10)
	var zipFileName string
	var zipf *zip.Writer
	if zipOpt {
		zipFileName = "/tmp/" + strconv.Itoa(RandId())
		zipfn, fErr := os.Create(zipFileName)
		if fErr != nil {
			return nil, zipFileInfo{}, fErr
		}
		zipf = zip.NewWriter(zipfn)
	}
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
		if zipOpt && !info.IsDir() {
			fErr = zipOne(zipf, path)
			if fErr != nil {
				lg.Println(fErr)
				return fErr
			}
		}
		return nil
	}
	fErr = filepath.Walk(base, WalkFunc)
	if fErr != nil {
		lg.Println(fErr)
		return nil, zipFileInfo{}, fErr
	}
	var zipFI = zipFileInfo{}
	if zipOpt {
		fErr = zipf.Close()
		if fErr != nil {
			return nil, zipFileInfo{}, fErr
		}
		var zipMd5 string
		zipMd5, fErr = Md5OfAFile(zipFileName)
		if fErr != nil {
			return nil, zipFileInfo{}, fErr
		}
		// zipMd5Map = make(map[string]string)
		// zipMd5Map[zipFileName] = zipMd5
		zipFI.name = zipFileName
		zipFI.md5s = zipMd5
	}
	return md5List, zipFI, nil
}

// func DefaultSync(mg *Message, targets []string) []*result {

// }
