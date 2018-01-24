package gosync

import (
	"os"
)

func localOP(slinkNeedCreat map[string]string, slinkNeedChange map[string]string, needDelete []string) error {
	lg.Println(os.Getwd())
	var err error
	for k, v := range slinkNeedCreat {
		err = os.Symlink(k, v)
		if err != nil {
			lg.Println(err)
			return err
		}
	}
	for k, v := range slinkNeedChange {
		err = os.Remove(k)
		if err != nil {
			lg.Println(err)
			return err
		}
		err = os.Symlink(k, v)
		if err != nil {
			lg.Println(err)
			return err
		}
	}
	if len(needDelete) == 0 {
		return nil
	}
	for _, v := range needDelete {
		err = os.Remove(v)
		if err != nil {
			lg.Println(v)
			lg.Println(err)
			return err
		}
	}
	return nil
}
