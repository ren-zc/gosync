package gosync

type Message struct {
	MgID      int    // reserved
	MgType    string // cmd,auth,file,info,task
	MgName    string // cmd name, auth username, file name, info name, task: DefaultSync/UpdateSync
	MgByte    []byte // file piece
	MgString  string // cmd option, autho user passwd, sync task target hosts
	IntOption int    // file piece number or other
	StrOption string // start, continue, end
	SrcPath   string // src file path or task src path
	DstPath   string // dst file path or task dst path
}
