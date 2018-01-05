package gosync

import (
	"bufio"
	"encoding/gob"
	"log"
)

func hdTask(mg *Message, cnRd *bufio.Reader, cnWt *bufio.Writer, dec *gob.Decoder, enc *gob.Encoder) {
	// defer conn.Close()
	switch mg.MgName {
	case "DefaultSync":
		// handle DefaultSync
	case "UpdateSync":
		// handle UpdateSync
	default:
		writeErrorMg("error, not a recognizable MgName.", cnWt, enc)
	}

}

func hdFile(mg *Message, cnRd *bufio.Reader, cnWt *bufio.Writer, dec *gob.Decoder, enc *gob.Encoder) {
	// defer conn.Close()
}

func hdNoType(mg *Message, cnRd *bufio.Reader, cnWt *bufio.Writer, dec *gob.Decoder, enc *gob.Encoder) {
	// defer conn.Close()
	writeErrorMg("error, not a recognizable message.", cnWt, enc)
}

func writeErrorMg(s string, cnWt *bufio.Writer, enc *gob.Encoder) {
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
