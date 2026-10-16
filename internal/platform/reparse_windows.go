//go:build windows

package platform

import (
	"errors"
	"golang.org/x/sys/windows"
)

func NoReparse(path string) error {
	p, e := windows.UTF16PtrFromString(path)
	if e != nil {
		return e
	}
	a, e := windows.GetFileAttributes(p)
	if e != nil {
		return e
	}
	if a&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 || a&windows.FILE_ATTRIBUTE_ENCRYPTED != 0 {
		return errors.New("reparse points and EFS files are not supported in vault data or backups")
	}
	return nil
}
