package platform

import (
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// The caller requires an explicit owner action. Never runs on ordinary startup.
func TrustLocalCertificate(path string) error {
	user, e := windows.GetCurrentProcessToken().GetTokenUser()
	if e != nil {
		return e
	}
	if user.User.Sid.IsWellKnown(windows.WinLocalSystemSid) || user.User.Sid.IsWellKnown(windows.WinLocalServiceSid) || user.User.Sid.IsWellKnown(windows.WinNetworkServiceSid) {
		return errors.New("run MyVault.exe trust-local --data YOUR_DATA_FOLDER from your Windows login, not the service account")
	}
	tool := filepath.Join(os.Getenv("SystemRoot"), "System32", "certutil.exe")
	command := exec.Command(tool, "-user", "-addstore", "Root", path)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	if output, e := command.CombinedOutput(); e != nil {
		return fmt.Errorf("Windows could not trust the local certificate: %s", output)
	}
	return nil
}
