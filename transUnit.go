package gosync

import (
	// "archive/zip"
	// "bufio"
	// "encoding/gob"
	"fmt"
	// "log"
	// "net"
	// "os"
	// "path/filepath"
	"sort"
	// "strconv"
	// "sync"
)

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

func (tu *transUnit) String() string {
	return fmt.Sprintf("%v\t%v\t%v\t%v\n", tu.hosts, tu.fileMd5List, tu.name, tu.md5s)
}

func getTransUnit(mg *Message, targets []string) (map[md5s]transUnit, error) {
	lg.Println("in getTransUnit") // ****** test ******

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
	sort.Strings(fileMd5List)
	listMd5 := Md5OfASlice(fileMd5List)
	TravHosts(targets, fileMd5List, md5s(listMd5), mg)
	di := diffInfo{}
	hostNum := len(hosts)
	for i := 0; i < hostNum; i++ {
		di = <-diffCh
		// lg.Println(di) // ****** test ******
		if di.files == nil {
			continue
		}
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
	lg.Println(tus) // ****** test ******
	return tus, nil
}
