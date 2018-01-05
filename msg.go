package gosync

type Message struct {
	MgID      int    // reserved
	MgType    string // cmd,auth,file,info,task
	MgName    string // cmd name, auth username, file name, info name, task: DefaultSync/UpdateSync
	MgByte    []byte // cmd option, autho user passwd, file piece, sync task target hosts
	MgString  string // same as above
	IntOption int    // file piece number or other
	StrOption string // start, continue, end
}
