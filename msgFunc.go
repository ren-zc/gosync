package gosync

import (
	"fmt"
	"github.com/jacenr/filediff/diff"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
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
		lg.Printf("%s\t%s\n", c.hostIP, c.Err)
		allConn[c.hostIP] = c.ret
		l = len(allConn)
		// 相等表示所有host已返回sync结果
		if l == i {
			break
		}
	}
	close(retCh)
	close(retReady)
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
		taskID := getTaskID()
		t.put(taskID)
		// lg.Println(t)
		for {
			if t.ask(taskID) {
				break
			}
			time.Sleep(1 * time.Second)
		}
		diffCh := make(chan diffInfo)
		hostNum := len(targets)

		// 等待同步结果, 从retReady接收"Done"表示allConn写入完成
		allConn := map[hostIP]ret{}   // 用于收集host返回的sync结果
		retCh := make(chan hostRet)   // 用于接收各host的sync结果
		retReady := make(chan string) // 从此channel读取到Done表示所有host已返回结果
		go cnMonitor(hostNum, allConn, retCh, retReady)

		// traHosts, 用于获取文件列表和同步结果
		fileMd5List, err := Traverse(mg.SrcPath)
		if err != nil {
			lg.Println(err)
			// 将tarErr以Message的形式发送给客户端
			// 待补充
			return
		}
		sort.Strings(fileMd5List)
		listMd5 := Md5OfASlice(fileMd5List)
		TravHosts(targets, fileMd5List, md5s(listMd5), mg, diffCh, retCh, taskID)

		// get transUnits
		tus, err := getTransUnit(mg.Zip, hostNum, diffCh, retCh)
		if err != nil {
			fmt.Println(err)
		}

		// test fmt
		for k, v := range tus {
			lg.Printf("%s\t", k)
			lg.Println(v.hosts)
		}

		// *** 对每个tu执行同步文件操作, 将最终结果push到retCh ***
		//

		// *** 将allConn返回给客户端 ***
		<-retReady
		var cr ClientRet
		cr.MgID = mg.MgID
		cr.MgType = "result"
		cr.M = allConn
		err = gbc.gobConnWt(cr)
		if err != nil {
			// *** 记录本地日志 ***
			return
		}

		// 返回任务开始前的目录
		err = os.Chdir(cwd)
		if err != nil {
			lg.Println(err)
		}

		// *** end taskID ***
		t.tEnd(taskID)

	case "exec":
		// 预留, 后期扩展;

	default:
		writeErrorMg(mg, "error, not a recognizable MgName.", gbc)
	}
}

func hdFile(mg *Message, gbc *gobConn) {
	// defer conn.Close()
}

func hdFileMd5List(mg *Message, gbc *gobConn) {
	var slinkNeedCreat = make(map[string]string)
	var slinkNeedChange = make(map[string]string)
	var needDelete = make([]string, 0)

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
	// lg.Println(t)
	defer t.tEnd(mg.TaskID)
	for {
		if t.ask(mg.TaskID) {
			break
		}
		ret.TaskID = mg.TaskID
		ret.MgID = mg.MgID
		ret.MgType = "live" // heartbeat
		err = gbc.gobConnWt(ret)
		// lg.Println("waiting...")
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
		if !ok && mg.Del {
			needDelete = append(needDelete, k)
		}
	}
	for k, v := range diffrmM {
		if strings.HasPrefix(v, "symbolLink&&") {
			slinkNeedCreat[k] = strings.TrimPrefix(v, "symbolLink&&")
			delete(diffrmM, k)
		}
	}

	// *** do symbol link change: slinkNeedChange ***
	// *** do symbol link create: slinkNeedCreat ***
	// *** do delete extra files: needDelete ***
	err = localOP(slinkNeedCreat, slinkNeedChange, needDelete)
	if err != nil {
		ret.TaskID = mg.TaskID
		ret.MgID = mg.MgID
		ret.MgType = "result"
		ret.MgString = "Local operation failed."
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

	// *** 阻塞直到, 从channel读取同步结果 ***
	hR := <-hostRetCh
	err = gbc.gobConnWt(hR)
	if err != nil {
		// *** 记录本地日志 ***
		return
	}
}

func hdNoType(mg *Message, gbc *gobConn) {
	// defer conn.Close()
	writeErrorMg(mg, "error, not a recognizable message.", gbc)
}

func writeErrorMg(mg *Message, s string, gbc *gobConn) {
	errmg := Message{}
	errmg.MgType = "info"
	errmg.MgString = s
	errmg.IntOption = mg.MgID
	sendErr := gbc.gobConnWt(errmg)
	if sendErr != nil {
		log.Println(sendErr)
	}
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
