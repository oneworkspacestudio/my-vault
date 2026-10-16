//go:build windows

package platform

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// A LocalSystem service must never execute the portable copy from an exFAT or
// ordinary-user-writable directory. Keep its supervisor in an administrator-
// controlled local NTFS directory, independently of private vault data.
func InstallServiceBinary(source, dataRoot string) (string, error) {
	if !windows.GetCurrentProcessToken().IsElevated() {
		return "", errors.New("Windows service installation requires an administrator PowerShell; portable run does not require elevation")
	}
	base, knownFolderErr := windows.KnownFolderPath(windows.FOLDERID_ProgramData, 0)
	if knownFolderErr != nil || !filepath.IsAbs(base) {
		return "", errors.New("Windows ProgramData is unavailable")
	}
	if _, e := CheckDataVolume(base); e != nil {
		return "", e
	}
	sum := sha256.Sum256([]byte(filepath.Clean(dataRoot)))
	folder := filepath.Join(base, "OneWorkspaceMyVaultServices", fmt.Sprintf("%x", sum[:6]))
	for current := folder; ; current = filepath.Dir(current) {
		if _, e := os.Lstat(current); e == nil {
			if e = NoReparse(current); e != nil {
				return "", e
			}
		} else if !os.IsNotExist(e) {
			return "", e
		}
		if filepath.Dir(current) == current {
			break
		}
	}
	sd, e := windows.SecurityDescriptorFromString("O:BAG:BAD:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;GRGX;;;BU)")
	if e != nil {
		return "", e
	}
	acl, _, e := sd.DACL()
	if e != nil {
		return "", e
	}
	admins, e := windows.StringToSid("S-1-5-32-544")
	if e != nil {
		return "", e
	}
	sa := &windows.SecurityAttributes{SecurityDescriptor: sd}
	sa.Length = uint32(unsafe.Sizeof(*sa))
	// Create each directory with its final owner/DACL atomically. Refuse a
	// pre-created ordinary-user-owned folder rather than adopting attacker state.
	for _, dir := range []string{filepath.Dir(folder), folder} {
		p, e := windows.UTF16PtrFromString(dir)
		if e != nil {
			return "", e
		}
		if e = windows.CreateDirectory(p, sa); e != nil && !errors.Is(e, windows.ERROR_ALREADY_EXISTS) {
			return "", e
		}
		if e = NoReparse(dir); e != nil {
			return "", e
		}
		current, e := windows.GetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION)
		if e != nil {
			return "", e
		}
		owner, _, e := current.Owner()
		if e != nil {
			return "", e
		}
		if !owner.IsWellKnown(windows.WinBuiltinAdministratorsSid) && !owner.IsWellKnown(windows.WinLocalSystemSid) {
			return "", errors.New("service binary folder has an unexpected owner; administrator repair is required")
		}
		if e = windows.SetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, admins, nil, acl, nil); e != nil {
			return "", e
		}
	}
	in, e := os.Open(source)
	if e != nil {
		return "", e
	}
	defer in.Close()
	tmp, e := os.CreateTemp(folder, ".install-")
	if e != nil {
		return "", e
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, e = io.Copy(tmp, in); e == nil {
		e = tmp.Sync()
	}
	ce := tmp.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return "", e
	}
	if e = windows.SetNamedSecurityInfo(name, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, admins, nil, acl, nil); e != nil {
		return "", e
	}
	target := filepath.Join(folder, "MyVault.exe")
	if _, e = os.Lstat(target); e == nil {
		if e = NoReparse(target); e != nil {
			return "", e
		}
	}
	if e = os.Rename(name, target); e != nil {
		return "", errors.New("cannot update the protected service executable; stop the existing My Vault service first")
	}
	return target, nil
}
