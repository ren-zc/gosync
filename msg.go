package gosync

type Message struct {
	TaskID    string
	MgID      int    // reserved
	MgType    string // cmd,auth,file,info,task
	MgName    string // cmd name, auth username, file name, info name, ** del: task: DefaultSync/UpdateSync**
	MgByte    []byte // file piece
	MgString  string // cmd option, autho user passwd, sync task target hosts
	MgStrings []string
	IntOption int    // file piece number or other
	StrOption string // start, continue, end
	SrcPath   string // src file path or task src path
	DstPath   string // dst file path or task dst path
	Del       bool   // whether should the not exist files in src be deleted.
	Zip       bool   // whether should files be compressed.
	Overwrt   bool   // whether the conflicted files be overwrited.
	B         bool   // other bool option
	M         map[hostIP]ret
}

// 定义一个小型的Message, 减小网络流量
type ClientRet struct {
	MgID   int
	MgType string
	M      map[hostIP]ret
}

// type filePiece struct {
// 	TaskID    string
// 	MgID      int
// 	MgType    string // "fileStream"
// 	MgName    string // 文件名
// 	MgByte    []byte // 文件内容切片
// 	IntOption int    // 切片序号
// 	MgString  string // 用于标识所有切片发送完成
// 	B         bool   // 是否是文件的最后一片
// 	Zip       bool   // 是否是元压缩文件(需解压缩)
// }
