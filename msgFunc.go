package gosync

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"github.com/jacenr/filediff/diff"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
)

// ip addr
type hostIP string

// host返回的请求文件列表
type diffInfo struct {
	md5s // diff文件列表的md5
	hostIP
	files []string // 需要更新的文件, 即diff文件列表
}

// 负责管理allConn, 使用retCh channel
func cnMonitor(i int, allConn map[hostIP]ret, retCh chan hostRet, retReady chan string) {
	var c hostRet
	var l int
	for {
		c = <-retCh
		lg.Println(c) // ****** test ******
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

// hd: handle

func hdTask(mg *Message, cnRd *bufio.Reader, cnWt *bufio.Writer, dec *gob.Decoder, enc *gob.Encoder) {
	lg.Println(mg) // ****** test ******

	var checkOk bool
	var targets []string
	if checkOk, targets = checkTargets(mg); !checkOk {
		writeErrorMg(mg, "error, not valid ip addr in MgString.", cnWt, enc)
	}
	lg.Println(targets) // ****** test ******

	switch mg.MgName {
	case "sync":
		diffCh := make(chan diffInfo)
		hostNum := len(targets)

		// 等待同步结果, 从retReady接收"Done"表示allConn写入完成
		allConn := map[hostIP]ret{}   // 用于收集host返回的sync结果
		retCh := make(chan hostRet)   // 用于接收各host的sync结果
		retReady := make(chan string) // 从此channel读取到Done表示所有host已返回结果
		go cnMonitor(hostNum, allConn, retCh, retReady)

		// traHosts, 用于获取文件列表和同步结果
		var fileMd5List []string
		var traErr error
		fileMd5List, traErr = Traverse(mg.SrcPath)
		if traErr != nil {
			lg.Println(traErr)
			// 将tarErr以Message的形式发送给客户端
			// 待补充
			return
		}
		sort.Strings(fileMd5List)
		listMd5 := Md5OfASlice(fileMd5List)
		TravHosts(targets, fileMd5List, md5s(listMd5), mg, diffCh, retCh)

		// get transUnits
		var tus = make(map[md5s]transUnit)
		tus, err := getTransUnit(mg.Zip, hostNum, diffCh, retCh)
		if err != nil {
			fmt.Println(err)
		}
		// test fmt
		for k, v := range tus {
			lg.Printf("%s\t", k)
			lg.Println(v)
		}
		err = os.Chdir(pwd)
		if err != nil {
			lg.Println(err)
		}

		// 整理allConn返回给客户端
		//

	default:
		writeErrorMg(mg, "error, not a recognizable MgName.", cnWt, enc)
	}

}

func hdFile(mg *Message, cnRd *bufio.Reader, cnWt *bufio.Writer, dec *gob.Decoder, enc *gob.Encoder) {
	// defer conn.Close()
}

func hdFileMd5List(mg *Message, cnRd *bufio.Reader, cnWt *bufio.Writer, dec *gob.Decoder, enc *gob.Encoder) {
	lg.Println("Conn established.")

	var slinkNeedCreat = make(map[string]string)
	var slinkNeedChange = make(map[string]string)
	var needDelete = make([]string, 1)

	var ret Message
	localFilesMd5, err := Traverse(mg.DstPath)
	if err != nil {
		ret.MgID = mg.MgID
		ret.MgType = "result"
		ret.MgString = "Traverse in target host failure"
		ret.b = false
		// encerr:= enc.Encode(ret) 暂时不考虑这种情况, 需配合源主机的超时机制
		enc.Encode(ret)
	}
	sort.Strings(localFilesMd5)
	diffrm, diffadd := diff.DiffOnly(mg.MgStrings, localFilesMd5)
	// 重组成map
	diffrmM := make(map[string]string)
	diffaddM := make(map[string]string)
	for _, v := range diffrm {
		s := strings.Split(v, ",,")
		if len(s) != 1 {
			diffrmM[s[0]] = s[1]
		}
	}
	for _, v := range diffadd {
		s := strings.Split(v, ",,")
		if len(s) != 1 {
			diffaddM[s[0]] = s[1]
		}
	}
	// 整理
	for k, _ := range diffaddM {
		v2, ok := diffrmM[k]
		if ok && !mg.Overwrt {
			delete(diffrmM, k)
		}
		if ok && mg.Overwrt {
			if strings.HasPrefix(v2, "symbolLink&&") {
				slinkNeedChange[k] = strings.TrimPrefix(v2, "symbolLink&&")
				delete(diffrmM, k)
			}
		}
		if !ok {
			needDelete = append(needDelete, k)
		}
	}
	for k, v := range diffrmM {
		if strings.HasPrefix(v, "symbolLink&&") {
			slinkNeedCreat[k] = strings.TrimPrefix(v, "symbolLink&&")
			delete(diffrmM, k)
		}
	}
	// needTrans = diffrmM

	// do request needTrans files
	transFiles := []string{}
	for k, _ := range diffrmM {
		transFiles = append(transFiles, k)
	}
	sort.Strings(transFiles)
	transFilesMd5 := Md5OfASlice(transFiles)
	ret.MgID = mg.MgID
	ret.MgStrings = transFiles
	ret.MgType = "diffOfFilesMd5List"
	ret.MgString = transFilesMd5 // for check, reserved
	// encerr:= enc.Encode(ret) 暂时不考虑这种情况, 需配合源主机的超时机制
	// fmt.Println(ret) // ****** test ******
	err = enc.Encode(ret)
	if err != nil {
		lg.Println(err)
	}
	err = cnWt.Flush()
	if err != nil {
		lg.Println(err)
	}

	// do symbol link change

	// do symbol link create

	// do delete extra files
}

func hdNoType(mg *Message, cnRd *bufio.Reader, cnWt *bufio.Writer, dec *gob.Decoder, enc *gob.Encoder) {
	// defer conn.Close()
	writeErrorMg(mg, "error, not a recognizable message.", cnWt, enc)
}

func writeErrorMg(mg *Message, s string, cnWt *bufio.Writer, enc *gob.Encoder) {
	errmg := Message{}
	errmg.MgType = "info"
	errmg.MgString = s
	errmg.IntOption = mg.MgID
	sendErr := enc.Encode(errmg)
	if sendErr != nil {
		log.Println(sendErr)
	}
	cnWt.Flush()
}

func checkTargets(mg *Message) (bool, []string) {
	targets := strings.Split(mg.MgString, ",")

	ipReg, regErr := regexp.Compile(`^(\d{1,3}\.){3}\d{1,3}$`)
	if regErr != nil {
		log.Println(regErr)
	}
	for _, v := range targets {
		if !ipReg.MatchString(v) {
			return false, nil
		}
	}
	return true, targets
}
