package gosync

import (
	"archive/zip"
	"bufio"
	// "fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

// **备用**, 对一个文件或一个目录进行压缩, 压缩文件放置于/tmp
// 压缩文件名为随机id.
// 之所以备用, 是因为还没有考虑清楚对哪些文件进行压缩, 是整体压缩
// 还是单个文件执行压缩.

// 暂定option: -z, 由用户指定是否进行压缩, 注意是整体打包压缩.

// 对f路径进行压缩
func Zipfiles(f string) (string, error) {
	fi, fiErr := os.Stat(f)
	if fiErr != nil {
		return "None", fiErr
	}
	var dir string
	var file string
	if fi.IsDir() {
		dir = f
	} else {
		dir = filepath.Dir(f)
		file = filepath.Base(f)
	}
	fiErr = os.Chdir(dir)
	if fiErr != nil {
		return "None", fiErr
	}
	zipFileName := "/tmp/" + strconv.Itoa(RandId())
	zipfn, crtErr := os.Create(zipFileName)
	if crtErr != nil {
		return "None", crtErr
	}
	zipf := zip.NewWriter(zipfn)

	WalkFunc := func(path string, info os.FileInfo, err error) error {
		// fmt.Println(path)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			zipErr := zipOne(zipf, path)
			if zipErr != nil {
				return zipErr
			}
		}
		return nil
	}

	var zipErr error
	// write zip
	if file != "" {
		zipErr = zipOne(zipf, file)
		if zipErr != nil {
			return "None", zipErr
		}
	}

	if file == "" {
		walkErr := filepath.Walk(".", WalkFunc)
		if walkErr != nil {
			return "None", walkErr
		}
	}

	// close zip
	zipCloseErr := zipf.Close()
	if zipCloseErr != nil {
		return "None", zipCloseErr
	}
	return zipFileName, nil
}

// 对FileList中的文件进行压缩
func zipFileList(files []string) (string, error) {
	zipFileName := "/tmp/" + strconv.Itoa(RandId())
	zipfn, crtErr := os.Create(zipFileName)
	if crtErr != nil {
		return "None", crtErr
	}
	zipf := zip.NewWriter(zipfn)
	for _, file := range files {
		if file != "" {
			zipErr := zipOne(zipf, file)
			if zipErr != nil {
				return "None", zipErr
			}

		}
	}
	// close zip
	zipCloseErr := zipf.Close()
	if zipCloseErr != nil {
		return "None", zipCloseErr
	}
	return zipFileName, nil
}

func zipOne(zipf *zip.Writer, f string) error {
	fz, err := zipf.Create(f)
	if err != nil {
		return err
	}
	fs, openErr := os.Open(f)
	defer fs.Close()
	if openErr != nil {
		return openErr
	}
	fb := bufio.NewReader(fs)
	var rdErr error
	var line []byte
	for {
		line, rdErr = fb.ReadBytes('\n')
		fz.Write(line)
		if rdErr != nil {
			if rdErr == io.EOF {
				break
			} else {
				return rdErr
			}
		}
	}
	return nil
}
