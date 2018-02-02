package gosync

// import (
// 	"sync"
// )

// 用于整理文件byte片的顺序
type filePieceBuf struct {
	// mu sync.Mutex
	m  map[string]map[int]*Message
	fs []string
	f  string
	// e  map[string]int
	i int // 要读取的下一个piece number
}

func newFpb() *filePieceBuf {
	fpb := new(filePieceBuf)
	// mu := sync.Mutex
	m := make(map[string]map[int]*Message)
	fs := make([]string, 0, 2)
	// e := make(map[string]int)
	// fpb.mu = mu
	fpb.m = m
	// fpb.e = e
	fpb.fs = fs
	return fpb
}

func (fpb *filePieceBuf) putFpb(mg Message) {
	// fpb.mu.Lock()
	// defer fpb.mu.Unlock()
	if fpb.m[mg.MgName] == nil {
		f := make(map[int]*Message)
		fpb.m[mg.MgName] = f
	}
	fpb.m[mg.MgName][mg.IntOption] = &mg
	fpb.fs = append(fpb.fs, mg.MgName)
}

func (fpb *filePieceBuf) getFpb() Message {
	// var mg *Message
	mg := &(Message{})
	// fpb.mu.Lock()
	// defer fpb.mu.Unlock()
	if fpb.f == "" {
		l := len(fpb.fs)
		if l == 0 {
			return *mg
		}
		if l == 1 {
			fpb.f = fpb.fs[0]
			fpb.fs = make([]string, 0, 2)
		}
		if l > 1 {
			fpb.f = fpb.fs[0]
			fpb.fs = fpb.fs[1:]
		}
	}
	fpb.i++
	var ok bool
	mg, ok = fpb.m[fpb.f][fpb.i]
	if !ok {
		fpb.i--
		return *mg
	}
	delete(fpb.m[fpb.f], fpb.i)
	if mg.B { // 当前f的最后一片
		delete(fpb.m, fpb.f)
		fpb.f = ""
		fpb.i = 0
	}
	return *mg
}

func fpbMonitor(fpb *filePieceBuf, putCh chan Message, getCh chan Message) {
	var mg1 Message
	var mg2 Message
	var ok bool
	var allPieces int
	var sendPieces int
	var n int
ENDFPBM:
	for {
		switch n {
		case 0: // 当putCh发送方确认文件传输任务完成, 就会关闭putCh, 那么ok=false
			n = 1
			mg1, ok = <-putCh
			// lg.Println(mg1)
			// lg.Println(ok)
			if ok {
				if mg1.MgString == "allEnd" {
					lg.Println(mg1)
					allPieces = mg1.IntOption
					lg.Println(allPieces)
					lg.Println("allPieces setted")
					getCh <- mg1
					continue ENDFPBM
				}
				// lg.Println("fpbMonitor get fileStream")
				// if !ok {
				// 	close(getCh)
				// 	lg.Println("getCh closed.")
				// 	break ENDFPBM
				// }
				lg.Println(mg1)
				fpb.putFpb(mg1)
				lg.Println("mg1 putted")
			}
		case 1:
			n = 0
			mg2 = fpb.getFpb()
			getCh <- mg2
			if mg2.MgType == "fileStream" {
				n = 1
				sendPieces++
				lg.Println(sendPieces)
				lg.Println(mg2)
			}
			// lg.Println("fpbMonitor send fileStream")
			lg.Println(sendPieces != 0 && sendPieces == allPieces)
			if sendPieces != 0 && sendPieces == allPieces {
				close(getCh)
				lg.Println("getCh closed")
				break ENDFPBM
			}
		}
		// select {
		// case mg1 = <-putCh: // 当putCh发送方确认文件传输任务完成, 就会关闭putCh, 那么mg1==nil
		// 	if mg1 == nil {
		// 		close(getCh)
		// 		break ENDFPBM
		// 	}
		// 	fpb.putFpb(mg1)
		// // case getCh <- mg2:
		// default:
		// 	mg2 = fpb.getFpb()
		// 	if mg2 != nil {
		// 		getCh <- mg2
		// 	}
		// }

	}
	lg.Println("fpbMonitor end")
}

/*
func fpbMonitor(fpb *filePieceBuf, putCh chan Message, getCh chan Message) {
	// for {
	// 	mg := <-putCh
	// 	lg.Println(mg)
	// 	getCh <- mg
	// }
	for v := range putCh {
		lg.Println(v)
		getCh <- v
	}
}
*/
