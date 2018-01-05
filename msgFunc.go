package gosync

import (
	"bufio"
	"encoding/gob"
	"log"
	"regexp"
	"strings"
)

func hdTask(mg *Message, cnRd *bufio.Reader, cnWt *bufio.Writer, dec *gob.Decoder, enc *gob.Encoder) {
	// defer conn.Close()
	switch mg.MgName {
	case "DefaultSync":
		// handle DefaultSync
		if !checkTargets(mg) {
			writeErrorMg(mg, "error, not valid ip addr in MgString.", cnWt, enc)
		}
	case "UpdateSync":
		// handle UpdateSync
		if !checkTargets(mg) {
			writeErrorMg(mg, "error, not valid ip addr in MgString.", cnWt, enc)
		}
	default:
		writeErrorMg(mg, "error, not a recognizable MgName.", cnWt, enc)
	}

}

func hdFile(mg *Message, cnRd *bufio.Reader, cnWt *bufio.Writer, dec *gob.Decoder, enc *gob.Encoder) {
	// defer conn.Close()
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

func checkTargets(mg *Message) bool {
	targets := strings.Split(mg.MgString, ",")
	ipReg, regErr := regexp.Compile(`^(\d{1,3}\.){3}\d{1,3}$`)
	if regErr != nil {
		log.Println(regErr)
	}
	for _, v := range targets {
		if !ipReg.MatchString(v) {
			return false
		}
	}
	return true
}
