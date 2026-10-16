//go:build windows

package platform

import (
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestServiceInstallRequiresElevationBeforeChanges(t *testing.T) {
	if windows.GetCurrentProcessToken().IsElevated() {
		t.Skip("negative non-admin check")
	}
	path, e := InstallServiceBinary("missing-source.exe", t.TempDir())
	if path != "" || e == nil || !strings.Contains(e.Error(), "administrator") {
		t.Fatal("service install did not fail before filesystem mutation")
	}
}
