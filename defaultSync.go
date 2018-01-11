package gosync

import (
	"archive/zip"
	// "fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
)

type ret struct {
	status bool
	err    error
}
type md5s string
type fileList []string
type ips []string
type cnRet struct {
	net.Conn
	ret
}

// var md5OfFileList = map[md5s]fileList{}
// var md5MapOfHost = map[md5s]ips{}
type zipFileInfo struct {
	name string
	md5  string
}
type transUnit struct {
	hosts       []string
	fileMd5List []string
	zipFileInfo
}

var allConn = map[net.Conn]ret{}
var retReady = make(chan string) // 从此channel读取到Done表示所有host已返回结果
var lg *log.Logger

func init() {
	lg = log.New(os.Stdout, "Err ", log.Lshortfile)
}

// 负责管理allConn
func cnMonitor(ch chan cnRet, i int) {
	var c = cnRet{}
	var l int
	for {
		c = <-ch
		allConn[c.Conn] = c.ret
		l = len(allConn)
		if l == i {
			// 相等表示所有host已返回结果
			break
		}
	}
	close(ch)
	retReady <- "Done"
}

func TravHosts(hosts []string, mg *Message, defaultSync bool) ([]transUnit, error) {
	// 准备src文件列表, 如果mg中zip选项为true, 则同时返回zip文件md5 map
	tus := make([]transUnit)
	var fileMd5List []string
	var zipFI zipFileInfo
	var traErr error
	if defaultSync {
		fileMd5List, zipFI, traErr = Traverse(mg.SrcPath, mg.Zip)
		tu := transUnit{}
		tu.hosts = hosts
		tu.fileMd5List = fileMd5List
		tu.zipFileInfo = zipFI
		tus = append(tus, tu)
		return tus, nil
	} else {
		// updateSync模式, 第一次生成file md5 list仅用于各host比对, 无须zip文件
		fileMd5List, _, traErr = Traverse(mg.SrcPath, false)
	}
	if traErr != nil {
		return traErr
	}

	// 启用conn监控goroutine
	retCh := make(chan cnRet)
	go cnMonitor(retCh, len(hosts))

	// 和所有host建立连接
	var conn net.Conn
	var cnErr error
	var port = "8999"
	for _, host := range hosts {
		conn, cnErr = net.Dial("tcp", host+port)
		if cnErr != nil {
			re := ret{false, cnErr}
			retCh <- re
			continue
		}
		// handle conn
		go hdRetConn(retCh, conn)
	}
}

func hdRetConn(ch chan cnRet, conn net.Conn) {
	defer conn.Close()
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
		zipFI.md5 = zipMd5
	}
	return md5List, zipFI, nil
}

// func DefaultSync(mg *Message, targets []string) []*result {

// }
