package gosync

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"sync"
)

var nullErr = fmt.Errorf("%s", "No target in list.")
var noResp = fmt.Errorf("%s\n", "No responses.")
var wg sync.WaitGroup

type ipStr string

type result struct {
	err  error
	info string
}

// type resultP *result
func initResult(err error, info string) *result {
	re := new(result)
	re.err = err
	re.info = info
	return re
}

func (re *result) String() {
	fmt.Printf("%s\t%s\n", re.info, re.err)
}

type targetSet struct {
	list    []string
	listLen int
	point   int
}

func initSet(s []string) *targetSet {
	ts := new(targetSet)
	ts.list = s
	ts.point = 0
}

// 使用monitor避开读写锁.
func (ts *targetSet) get() (string, error) {
	p := ts.point
	if p == ts.listLen {
		return "", nullErr
	}
	ts.point += 1
	return ts.list[p], nil
}

// 拒绝读写锁.
func tsMonitor(ts *targetSet, ipCh chan ipStr) {
	var ipOne string
	var ipErr error
	for {
		ipOne, ipErr = ts.get()
		if ipErr != nil {
			close(ipCh)
			break
		}
		ipCh <- ipStr(ipOne)
	}
}

func syncFile(mg *Message, src string, ipCh chan ipStr, reCh chan *result) {
	defer wg.Done()
	port := "8999"
	for {
		ip, ok := string(<-ipCh)
		if !ok {
			break
		}
		conn, cnErr := net.Dial("tcp", ip+port)
		// *** PAUSE HERE, 至此, 新建alg2分支, 使用新的算法 ***
	}
}

func DefaultSync(mg *Message, targets []string) []*result {
	var res = []*result{}
	src := mg.SrcPath

	ts := initSet(targets)
	ipCh := make(chan ipStr)
	go tsMonitor(ts, ipCh)

	var compErr error
	// var zipFileName string
	if mg.Zip {
		src, compErr = Zipfiles(src)
	}
	if compErr != nil {
		re := initResult(compErr, "local")
		res = append(res, re)
		return res
	}

	// goroutines number.
	var goNumber = 4
	reCh := make(chan *result)
	for i := 0; i < goNumber; i++ {
		wg.Add(1)
		go syncFile(mg, src, ipCh, reCh)
	}

	go func() {
		wg.Wait()
		close(reCh)
	}()

	for re := range reCh {
		res = append(res, re)
	}

	return res
}
