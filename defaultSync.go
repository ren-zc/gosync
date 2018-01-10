package gosync

import (
	"archive/zip"
	// "fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

var lg *log.Logger

func init() {
	lg = log.New(os.Stdout, "Err ", log.Lshortfile)
}

func Traverse(path string, zipOpt bool) ([]string, map[string]string, error) {
	f, fErr := os.Lstat(path)
	if fErr != nil {
		return nil, nil, fErr
	}
	var dir string
	var base string
	if f.IsDir() {
		dir = path
		base = "."
	} else {
		dir = filepath.Dir(path)
		base = filepath.Base(path)
	}
	fErr = os.Chdir(dir)
	if fErr != nil {
		return nil, nil, fErr
	}
	md5List := make([]string, 10)
	var zipFileName string
	var zipf *zip.Writer
	if zipOpt {
		zipFileName = "/tmp/" + strconv.Itoa(RandId())
		zipfn, fErr := os.Create(zipFileName)
		if fErr != nil {
			return nil, nil, fErr
		}
		zipf = zip.NewWriter(zipfn)
	}
	var md5Str string
	WalkFunc := func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			md5Str, fErr = Md5OfAFile(path)
			if fErr != nil {
				lg.Println(fErr)
				return fErr
			}
		}
		md5Str = path + "," + md5Str
		md5List = append(md5List, md5Str)
		if zipOpt && !info.IsDir() {
			fErr = zipOne(zipf, path)
			if fErr != nil {
				lg.Println(fErr)
				return fErr
			}
		}
		return nil
	}
	fErr = filepath.Walk(base, WalkFunc)
	if fErr != nil {
		lg.Println(fErr)
		return nil, nil, fErr
	}
	var zipMd5Map map[string]string
	if zipOpt {
		fErr = zipf.Close()
		if fErr != nil {
			return nil, nil, fErr
		}
		var zipMd5 string
		zipMd5, fErr = Md5OfAFile(zipFileName)
		if fErr != nil {
			return nil, nil, fErr
		}
		zipMd5Map = make(map[string]string)
		zipMd5Map[zipFileName] = zipMd5
	}
	return md5List, zipMd5Map, nil
}

// func DefaultSync(mg *Message, targets []string) []*result {

// }
