package vault

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecoveryGuideUsesSecureRestoreOrigin(t *testing.T) {
	w := httptest.NewRecorder()
	serveRecovery(w, httptest.NewRequest("GET", "https://localhost:8915/_myvault/recovery", nil), "https://localhost:8915")
	if !strings.Contains(w.Body.String(), "--origin https://localhost:8915") || strings.Contains(w.Body.String(), "--origin http://") {
		t.Fatal("recovery guide must use an origin accepted by the HTTPS-only client")
	}
}
