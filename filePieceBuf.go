package gosync

// 用于整理文件byte片的顺序
type filePieceBuf struct {
	m  map[string]map[int]*Message
	fs []string
	f  string
	i  int // 要读取的下一个piece number
}

func newFpb() *filePieceBuf {
	fpb := new(filePieceBuf)
	m := make(map[string]map[int]*Message)
	fs := make([]string, 0, 2)
	fpb.m = m
	fpb.fs = fs
	return fpb
}

func (fpb *filePieceBuf) putFpb(mg Message) {
	if fpb.m[mg.MgName] == nil {
		f := make(map[int]*Message)
		fpb.m[mg.MgName] = f
	}
	fpb.m[mg.MgName][mg.IntOption] = &mg
	fpb.fs = append(fpb.fs, mg.MgName)
	DubugInfor("putted ", fpb.m[mg.MgName][mg.IntOption])
}

func (fpb *filePieceBuf) getFpb() Message {
	// mg := &(Message{})
	if fpb.f == "" {
		l := len(fpb.fs)
		if l == 0 {
			return Message{}
		}
		if l == 1 {
			fpb.f = fpb.fs[0]
			fpb.fs = fpb.fs[:0]
		}
		if l > 1 {
			fpb.f = fpb.fs[0]
			fpb.fs = fpb.fs[1:]
		}
	}
	fpb.i++
	// var ok bool
	mg, ok := fpb.m[fpb.f][fpb.i]
	if !ok {
		fpb.i--
		DubugInfor("get failed.")
		return Message{}
	}
	delete(fpb.m[fpb.f], fpb.i)
	if mg.B { // 当前f的最后一片
		delete(fpb.m, fpb.f)
		fpb.f = ""
		fpb.i = 0
	}
	DubugInfor("get ", mg)
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
			if ok {
				if mg1.MgString == "allEnd" {
					DubugInfor(mg1)
					allPieces = mg1.IntOption
					DubugInfor(allPieces)
					DubugInfor("allPieces setted")
					getCh <- mg1
					continue ENDFPBM
				}
				DubugInfor(mg1)
				fpb.putFpb(mg1)
				DubugInfor("mg1 putted")
			}
		case 1:
			// DubugInfor(sendPieces != 0 && sendPieces == allPieces)
			if sendPieces != 0 && sendPieces == allPieces {
				close(getCh)
				DubugInfor("getCh closed")
				break ENDFPBM
			}
			n = 0
			mg2 = fpb.getFpb()
			if mg2.MgType != "fileStream" {
				continue ENDFPBM
			}
			getCh <- mg2
			DubugInfor(mg2)
			if mg2.MgType == "fileStream" {
				n = 1
				sendPieces++
				DubugInfor(sendPieces)
				// DubugInfor(mg2)
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
	// lg.Println("fpbMonitor end")
	DubugInfor("fpbMonitor end")
}

/* 简单测试
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
