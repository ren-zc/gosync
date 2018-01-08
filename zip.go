package gosync

import (
	"archive/zip"
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

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
	os.Chdir(dir)
	zipFileName := strconv.Itoa(RandId())
	zipfn, crtErr := os.Create(zipFileName)
	if crtErr != nil {
		return "None", crtErr
	}
	zipf := zip.NewWriter(zipfn)

	WalkFunc := func(path string, info os.FileInfo, err error) error {
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

func zipOne(zipf *zip.Writer, f string) error {
	fz, err := zipf.Create(f)
	if err != nil {
		return err
	}
	fs, openErr := os.Open(f)
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
