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

// hd: handle

func hdTask(mg *Message, cnRd *bufio.Reader, cnWt *bufio.Writer, dec *gob.Decoder, enc *gob.Encoder) {
	lg.Println(mg) // ****** test ******

	var checkOk bool
	var targets []string
	switch mg.MgName {
	case "sync":
		if checkOk, targets = checkTargets(mg); !checkOk {
			writeErrorMg(mg, "error, not valid ip addr in MgString.", cnWt, enc)
		}

		lg.Println(targets) // ****** test ******

		// get transUnits
		var tus = make(map[md5s]transUnit)
		tus, err := getTransUnit(mg, targets)
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
	default:
		writeErrorMg(mg, "error, not a recognizable MgName.", cnWt, enc)
	}

}

func hdFile(mg *Message, cnRd *bufio.Reader, cnWt *bufio.Writer, dec *gob.Decoder, enc *gob.Encoder) {
	// defer conn.Close()
}

// var needTrans map[string]string
var slinkNeedCreat = make(map[string]string)
var slinkNeedChange = make(map[string]string)
var needDelete = make([]string, 1)

func hdFileMd5List(mg *Message, cnRd *bufio.Reader, cnWt *bufio.Writer, dec *gob.Decoder, enc *gob.Encoder) {
	lg.Println("Conn established.")

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

	// ****** test ******
	for _, v := range targets {
		lg.Printf("ip: %s", v)
	}

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
