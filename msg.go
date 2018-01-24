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

type ClientRet struct {
	MgID   int
	MgType string
	M      map[hostIP]ret
}
