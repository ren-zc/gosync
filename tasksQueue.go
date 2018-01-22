package gosync

import (
	"strconv"
	"sync"
)

type State bool

var idNumber int

const (
	Running   = true
	Complated = false
)

type Tasks struct {
	Queue   []string
	Current string
	Status  State
}

var taskMu sync.Mutex
var t *Tasks

// 把自己的task id put到队列
func (t *Tasks) put(taskeID string) {
	taskMu.Lock()
	defer taskMu.Unlock()
	if t.Current == "" {
		t.Current = taskeID
		return
	}
	t.Queue = append(t.Queue, taskeID)
}

// 查询自己是否可以执行, false为不可执行
func (t *Tasks) ask(taskID string) bool {
	if t.Status == Running {
		return false
	}
	b := t.Current == taskID
	if b {
		taskMu.Lock()
		defer taskMu.Unlock()
		t.Status = Running
	}
	return b
}

// task执行完成时, 把status置为false, 从队列中取出第一个元素赋值给current
// 同时将元素从队列中删除
func (t *Tasks) tEnd(taskID string) {
	taskMu.Lock()
	defer taskMu.Unlock()
	t.Status = Complated
	l := len(t.Queue)
	if l == 0 {
		t.Current = ""
		return
	}
	if l == 1 {
		t.Current = t.Queue[0]
		t.Queue = []string{}
		return
	}
	if l > 1 {
		t.Current = t.Queue[0]
		t.Queue = t.Queue[1:]
		return
	}
}

func getTaskID(ipStr string) string {
	taskId := ipStr + "." + strconv.Itoa(idNumber)
	idNumber++
	return taskId
}
