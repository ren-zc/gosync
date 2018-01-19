package gosync

import (
	"fmt"
	// "sort"
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

func getTransUnit(zip bool, hostNum int, diffCh chan diffInfo, retCh chan hostRet) (map[md5s]transUnit, error) {
	lg.Println("in getTransUnit") // ****** test ******

	// hosts := make([]hostIP, 0)
	// for _, ipString := range targets {
	// 	hosts = append(hosts, hostIP(ipString))
	// }
	var tus = make(map[md5s]transUnit)
	// var zipFI zipFileInfo
	// hostNum := len(hosts)

	di := diffInfo{}
	for i := 0; i < hostNum; i++ {
		di = <-diffCh
		if di.files == nil {
			continue
		}
		if len(di.files) == 0 {
			putRetCh(di.hostIP, nil, retCh)
			continue
		}
		tu, ok := tus[di.md5s]
		if ok {
			tu.hosts = append(tu.hosts, di.hostIP)
			// lg.Println(tu.hosts)
		} else {
			h := []hostIP{}
			h = append(h, di.hostIP)
			lg.Println(h) // ****** test ******
			tu.hosts = h
			tu.fileMd5List = di.files
			if zip {
				zipName, zipMd5, traErr := Md5AndZip(di.files)
				if traErr != nil {
					return nil, traErr
				}
				zipFI := zipFileInfo{}
				zipFI.name = zipName
				zipFI.md5s = zipMd5
			}
			tu.zipFileInfo = zipFI
		}
		tus[di.md5s] = tu
	}
	// 返回tus, 即一个传输任务单元
	return tus, nil
}
