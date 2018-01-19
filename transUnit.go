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
	var tus = make(map[md5s]transUnit)

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
		} else {
			h := []hostIP{}
			h = append(h, di.hostIP)
			tu.hosts = h
			tu.fileMd5List = di.files
			zipFI := zipFileInfo{}
			if zip {
				zipName, zipMd5, traErr := Md5AndZip(di.files)
				if traErr != nil {
					return nil, traErr
				}
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
