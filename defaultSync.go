package gosync

import (
	"fmt"
	// "os"
)

var noResp = fmt.Errorf("%s\n", "No responses.")

func DefaultSync(src, dst string, targets []string) (error, string) {
	zipfileName, compErr := Zipfiles(src)
	if compErr != nil {
		return compErr, "error"
	}
	return noResp, "error"
}
