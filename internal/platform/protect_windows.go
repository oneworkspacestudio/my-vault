//go:build windows

package platform

import (
	"fmt"
	"golang.org/x/sys/windows"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"unsafe"
)

type DataVolume struct {
	Root       string `json:"root"`
	FileSystem string `json:"fileSystem"`
}

// CheckDataVolume is read-only and supports folders which do not exist yet.
// Setup calls it before creating files or starting any installation work.
func CheckDataVolume(dir string) (DataVolume, error) {
	var result DataVolume
	abs, e := filepath.Abs(dir)
	if e != nil {
		return result, e
	}
	probe := abs
	for {
		info, err := os.Stat(probe)
		if err == nil {
			if !info.IsDir() {
				return result, fmt.Errorf("choose a folder, not a file")
			}
			break
		}
		if !os.IsNotExist(err) {
			return result, fmt.Errorf("cannot access the selected folder: %w", err)
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return result, fmt.Errorf("the selected drive is not available")
		}
		probe = parent
	}
	path, e := windows.UTF16PtrFromString(probe)
	if e != nil {
		return result, e
	}
	root := make([]uint16, 32768)
	if e = windows.GetVolumePathName(path, &root[0], uint32(len(root))); e != nil {
		return result, fmt.Errorf("cannot identify the selected drive: %w", e)
	}
	var flags uint32
	fileSystem := make([]uint16, 260)
	if e = windows.GetVolumeInformation(&root[0], nil, 0, nil, nil, &flags, &fileSystem[0], uint32(len(fileSystem))); e != nil {
		return result, fmt.Errorf("cannot read the selected drive's storage format: %w", e)
	}
	result = DataVolume{Root: windows.UTF16ToString(root), FileSystem: windows.UTF16ToString(fileSystem)}
	if windows.GetDriveType(&root[0]) == windows.DRIVE_REMOTE {
		return result, fmt.Errorf("choose a folder on a local NTFS drive; network drives cannot hold Workspace's protected account data")
	}
	if flags&windows.FILE_PERSISTENT_ACLS == 0 {
		return result, fmt.Errorf("%s uses %s and cannot protect Workspace's account data. Choose a folder on an NTFS drive, such as C:", result.Root, result.FileSystem)
	}
	return result, nil
}

func ProtectDir(dir string) error {
	for current := filepath.Clean(dir); ; current = filepath.Dir(current) {
		if _, e := os.Lstat(current); e == nil {
			if e = NoReparse(current); e != nil {
				return e
			}
		}
		if filepath.Dir(current) == current {
			break
		}
	}
	if _, e := CheckDataVolume(dir); e != nil {
		return e
	}
	u, e := user.Current()
	if e != nil {
		return e
	}
	// Preserve the configuring user's access when the SYSTEM service restarts.
	ownerSID := u.Uid
	ownerPath := filepath.Join(dir, ".owner-sid")
	if info, err := os.Lstat(ownerPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("owner marker must not be a symbolic link")
	}
	if b, err := os.ReadFile(ownerPath); err == nil {
		ownerSID = strings.TrimSpace(string(b))
		if _, err = windows.StringToSid(ownerSID); err != nil {
			return fmt.Errorf("invalid data owner marker")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	// Replace the complete DACL, including pre-existing explicit grants.
	// Machine DPAPI requires this boundary on every existing child as well.
	sd, e := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;" + u.Uid + ")(A;OICI;FA;;;" + ownerSID + ")(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)")
	if e != nil {
		return e
	}
	acl, _, e := sd.DACL()
	if e != nil {
		return e
	}
	if e := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if e := NoReparse(path); e != nil {
			return e
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("data directory contains a symbolic link: %s", path)
		}
		if e := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); e != nil {
			return fmt.Errorf("cannot protect data permissions: %w", e)
		}
		return nil
	}); e != nil {
		return e
	}
	if _, err := os.Stat(ownerPath); os.IsNotExist(err) {
		return AtomicWrite(ownerPath, []byte(ownerSID))
	}
	return nil
}
func Protect(b []byte) ([]byte, error) {
	in := windows.DataBlob{Size: uint32(len(b))}
	if len(b) > 0 {
		in.Data = &b[0]
	}
	var out windows.DataBlob
	// Machine binding lets the configured SYSTEM service read credentials after an interactive setup.
	// Restrictive directory ACLs remain mandatory because other local users could decrypt an accessible blob.
	if e := windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN|windows.CRYPTPROTECT_LOCAL_MACHINE, &out); e != nil {
		return nil, e
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}
func Unprotect(b []byte) ([]byte, error) {
	in := windows.DataBlob{Size: uint32(len(b))}
	if len(b) > 0 {
		in.Data = &b[0]
	}
	var out windows.DataBlob
	if e := windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); e != nil {
		return nil, e
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}
