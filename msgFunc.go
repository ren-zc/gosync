package gosync

import (
	"github.com/jacenr/filediff/diff"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

var goTus int

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
		DubugInfor(c.hostIP, "\t", c.ErrStr)
		allConn[c.hostIP] = c.ret
		l = len(allConn)
		// 相等表示所有host已返回sync结果
		if l == i {
			break
		}
	}
	close(retCh)
	close(retReady)

	DubugInfor("retReady channel closed.")
}

// hd: handle

func hdTask(mg *Message, gbc *gobConn) {
	var checkOk bool
	var targets []string
	if checkOk, targets = checkTargets(mg); !checkOk {
		writeErrorMg(mg, "error, not valid ip addr in MgString.", gbc)
	}

	switch mg.MgName {
	case "sync":
		fileMd5List, err := Traverse(mg.SrcPath)
		if err != nil {
			PrintInfor(err)
			// 将tarErr以Message的形式发送给客户端, 待补充
			return
		}
		DubugInfor(len(fileMd5List))
		if len(fileMd5List) == 11 {
			var cr Message
			cr.MgID = mg.MgID
			cr.MgType = "result"
			// cr.M = allConn
			cr.MgString = "Please check the src directory, it must not be empty."
			DubugInfor(cr)
			err = gbc.gobConnWt(cr)
			if err != nil {
				// *** 记录本地日志 ***
				PrintInfor(err)
				return
			}
			return
		}
		taskID := getTaskID()
		t.put(taskID)
		defer t.end(taskID)
		for {
			if t.ask(taskID) {
				break
			}
			time.Sleep(1 * time.Second)
		}
		DubugInfor(t)
		diffCh := make(chan diffInfo)
		hostNum := len(targets)

		// 等待同步结果, 从retReady接收"Done"表示allConn写入完成
		allConn := map[hostIP]ret{}   // 用于收集host返回的sync结果
		retCh := make(chan hostRet)   // 用于接收各host的sync结果
		retReady := make(chan string) // 从此channel读取到Done表示所有host已返回结果
		go cnMonitor(hostNum, allConn, retCh, retReady)

		sort.Strings(fileMd5List)
		listMd5 := Md5OfASlice(fileMd5List)
		TravHosts(targets, fileMd5List, md5s(listMd5), mg, diffCh, retCh, taskID)

		// get transUnits
		tus, err := getTransUnit(mg.Zip, hostNum, diffCh, retCh)
		if err != nil {
			PrintInfor(err)
		}

		// 执行goTus个goroutine对每个tu执行同步文件操作, 将最终结果push到retCh
		zips := []string{}
		gos := make(chan struct{}, goTus)
		for i := 0; i < goTus; i++ {
			gos <- struct{}{}
		}
		for m, tu := range tus {
			DubugInfor(m, "\t", tu.hosts)
			// tranFile(m, &tu)
			go func(m md5s, tu transUnit) {
				st := <-gos
				tranFile(m, &tu)
				DubugInfor("a tu on going")
				gos <- st
			}(m, tu)

			// 收集zip文件名, 待task完成后删除其中的zip文件
			if mg.Zip {
				zips = append(zips, tu.zipFileInfo.name)
			}
		}

		// *** 将allConn返回给客户端 ***
		<-retReady
		var cr ClientRet
		cr.MgID = mg.MgID
		cr.MgType = "result"
		cr.M = allConn
		DubugInfor(cr)
		err = gbc.gobConnWt(cr)
		if err != nil {
			// *** 记录本地日志 ***
			PrintInfor(err)
			return
		}
		DubugInfor("the result returned to the client.")

		// 清理临时生成的zip文件
		for _, zip := range zips {
			DubugInfor(zip, ", this zip file will be deleted.")
			err = os.Remove(zip)
			if err != nil {
				PrintInfor(zip, "delete fail, ", err)
			}
		}

		close(gos)

	case "exec":
		// 预留, 后期扩展;

	default:
		writeErrorMg(mg, "error, not a recognizable MgName.", gbc)
	}
}

func hdFileMd5List(mg *Message, gbc *gobConn) {
	var slinkNeedCreat = make(map[string]string)
	var slinkNeedChange = make(map[string]string)
	var needDelete = make([]string, 0)
	var needCreDir = make([]string, 0)

	var ret Message
	var err error

	// 本机无须通过网络同步文件
	if mg.TaskID == t.Current && t.Status == Running {
		ret.TaskID = mg.TaskID
		ret.MgID = mg.MgID
		ret.MgType = "result"
		ret.MgString = "The src and dst on the same host."
		ret.B = false
		err = gbc.gobConnWt(ret)
		if err != nil {
			// *** 记录本地日志 ***
		}
		return
	}

	t.put(mg.TaskID)
	DubugInfor(t)
	defer t.end(mg.TaskID)
	for {
		if t.ask(mg.TaskID) {
			break
		}
		ret.TaskID = mg.TaskID
		ret.MgID = mg.MgID
		ret.MgType = "live" // heartbeat
		err = gbc.gobConnWt(ret)
		if err != nil {
			// *** 记录本地日志 ***
		}
		time.Sleep(1 * time.Second)
	}

	// 遍历本地目标路径失败
	localFilesMd5, err := Traverse(mg.DstPath)
	if err != nil {
		ret.TaskID = mg.TaskID
		ret.MgID = mg.MgID
		ret.MgType = "result"
		ret.MgString = "Traverse in target host failure"
		ret.B = false
		err = gbc.gobConnWt(ret)
		if err != nil {
			// *** 记录本地日志 ***
		}
		return
	}

	sort.Strings(localFilesMd5)

	var diffrm []string
	var diffadd []string

	if len(localFilesMd5) != 0 {
		diffrm, diffadd = diff.DiffOnly(mg.MgStrings, localFilesMd5)
	} else {
		diffrm, diffadd = mg.MgStrings, localFilesMd5
	}

	if len(diffrm) == 0 && len(diffadd) == 0 {
		var hR Message
		hR.MgType = "result"
		hR.B = false
		hR.MgString = "No file need be transfored."
		hostRetCh <- hR
		return
	}

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

	DubugInfor("diffrmM")
	DubugInfor(diffrmM)
	DubugInfor("diffaddM")
	DubugInfor(diffaddM)

	DubugInfor("overwrite")
	DubugInfor(mg.Overwrt)
	DubugInfor("delete")
	DubugInfor(mg.Del)

	// 整理
	for k, _ := range diffaddM {
		DubugInfor(k)
		v2, ok := diffrmM[k]
		if ok {
			if !mg.Overwrt {
				DubugInfor(k, " will not be trans-ed.")
				delete(diffrmM, k)
			}
			if mg.Overwrt {
				if strings.HasPrefix(v2, "symbolLink&&") {
					slinkNeedChange[k] = strings.TrimPrefix(v2, "symbolLink&&")
					delete(diffrmM, k)
				}
				DubugInfor(k, " will be deleted.")
				needDelete = append(needDelete, k)
			}
		}
		if !ok && mg.Del {
			DubugInfor(k, " will be deleted.")
			needDelete = append(needDelete, k)
		}

	}
	for k, v := range diffrmM {
		if strings.HasPrefix(v, "symbolLink&&") {
			slinkNeedCreat[k] = strings.TrimPrefix(v, "symbolLink&&")
			delete(diffrmM, k)
			continue
		}
		if v == "Directory" {
			needCreDir = append(needCreDir, k)
			delete(diffrmM, k)
		}
	}

	transFilesAndMd5 = diffrmM

	DubugInfor("diffrmM")
	DubugInfor(diffrmM)

	DubugInfor("slinkNeedCreat")
	DubugInfor(slinkNeedCreat)
	DubugInfor("slinkNeedChange")
	DubugInfor(slinkNeedChange)
	DubugInfor("needDelete")
	DubugInfor(needDelete)
	DubugInfor("needCreDir")
	DubugInfor(needCreDir)

	// 接收新文件的本地操作
	err = localOP(slinkNeedCreat, slinkNeedChange, needDelete, needCreDir)
	if err != nil {
		ret.TaskID = mg.TaskID
		ret.MgID = mg.MgID
		ret.MgType = "result"
		ret.MgString = err.Error()
		ret.B = false
		err = gbc.gobConnWt(ret)
		if err != nil {
			// *** 记录本地日志 ***
			// *** 待改进: 回滚操作或者提示哪些文件已被修改 ***
		}
		return
	}

	// do request needTrans files
	transFiles := []string{}
	for k, _ := range diffrmM {
		transFiles = append(transFiles, k)
	}

	DubugInfor(transFiles)

	sort.Strings(transFiles)
	transFilesMd5 := Md5OfASlice(transFiles)
	ret.TaskID = mg.TaskID
	ret.MgID = mg.MgID
	ret.MgStrings = transFiles
	ret.MgType = "diffOfFilesMd5List"
	ret.MgString = transFilesMd5
	err = gbc.gobConnWt(ret)
	if err != nil {
		// *** 记录本地日志 ***
		return
	}
}

func hdNoType(mg *Message, gbc *gobConn) {
	writeErrorMg(mg, "error, not a recognizable message.", gbc)
}

func writeErrorMg(mg *Message, s string, gbc *gobConn) {
	errmg := Message{}
	errmg.MgType = "info"
	errmg.MgString = s
	errmg.IntOption = mg.MgID
	sendErr := gbc.gobConnWt(errmg)
	if sendErr != nil {
		PrintInfor(sendErr)
	}
}

func checkTargets(mg *Message) (bool, []string) {
	targets := strings.Split(mg.MgString, ",")

	ipReg, regErr := regexp.Compile(`^(\d{1,3}\.){3}\d{1,3}$`)
	if regErr != nil {
		PrintInfor(regErr)
	}
	for _, v := range targets {
		if !ipReg.MatchString(v) {
			return false, nil
		}
	}
	return true, targets
}
