package VirtualRouterServer

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/neko233-com/virtual-router-go/internal/config"
)

func TestHandleUpdateAdminPassword_SuccessAndPersist(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd error: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	h := &HttpServer{cfg: &config.RouterServerConfig{RouterServerPort: 9999, HTTPMonitorPort: 19999, AdminPassword: "old-pass"}}
	body := map[string]string{"oldPassword": "old-pass", "newPassword": "new-pass"}
	payload, _ := json.Marshal(body)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/system/admin-password", bytes.NewReader(payload))
	h.handleUpdateAdminPassword(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rr.Code, rr.Body.String())
	}
	if h.cfg.AdminPassword != "new-pass" {
		t.Fatalf("expected in-memory password updated")
	}

	cfgPath := filepath.Join(tmp, config.RouterServerConfigName)
	cfgBytes, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config file error: %v", err)
	}
	var persisted config.RouterServerConfig
	if err := json.Unmarshal(cfgBytes, &persisted); err != nil {
		t.Fatalf("unmarshal persisted config error: %v", err)
	}
	if persisted.AdminPassword != "new-pass" {
		t.Fatalf("expected persisted password new-pass, got %q", persisted.AdminPassword)
	}
}
