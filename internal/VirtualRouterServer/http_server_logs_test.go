package VirtualRouterServer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleLogs_ReturnsRecentLinesByLimit(t *testing.T) {
	globalLogs.mu.Lock()
	backupLines := append([]string(nil), globalLogs.lines...)
	backupCapacity := globalLogs.capacity
	globalLogs.lines = []string{"l1", "l2", "l3"}
	globalLogs.capacity = 10
	globalLogs.mu.Unlock()

	t.Cleanup(func() {
		globalLogs.mu.Lock()
		globalLogs.lines = backupLines
		globalLogs.capacity = backupCapacity
		globalLogs.mu.Unlock()
	})

	h := &HttpServer{}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/logs?limit=2", nil)
	h.handleLogs(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Lines []string `json:"lines"`
			Count int      `json:"count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response error: %v", err)
	}

	if !resp.Success {
		t.Fatalf("expected success=true")
	}
	if resp.Data.Count != 2 {
		t.Fatalf("expected count=2, got %d", resp.Data.Count)
	}
	if len(resp.Data.Lines) != 2 || resp.Data.Lines[0] != "l2" || resp.Data.Lines[1] != "l3" {
		t.Fatalf("unexpected lines: %#v", resp.Data.Lines)
	}
}
