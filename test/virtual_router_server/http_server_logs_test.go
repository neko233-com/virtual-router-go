package virtual_router_server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	server "github.com/neko233-com/virtual-router-go/internal/VirtualRouterServer"
	"github.com/neko233-com/virtual-router-go/internal/config"
)

func TestHandleLogs_ReturnsRecentLinesByLimit(t *testing.T) {
	restore := server.SetProcessLogsForTest([]string{"l1", "l2", "l3"}, 10)
	t.Cleanup(restore)

	h := server.NewHttpServer(&config.RouterServerConfig{}, nil)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/logs?limit=2", nil)
	h.HandleLogsForTest(rr, req)

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

func TestHandleLogs_FilterByLevel(t *testing.T) {
	restore := server.SetProcessLogsForTest([]string{
		"2026-01-01 10:00:00 info server started",
		"2026-01-01 10:00:01 warn rpc timeout",
		"2026-01-01 10:00:02 error decode failed",
	}, 10)
	t.Cleanup(restore)

	h := server.NewHttpServer(&config.RouterServerConfig{}, nil)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/logs?limit=10&level=error", nil)
	h.HandleLogsForTest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Lines []string `json:"lines"`
			Count int      `json:"count"`
			Level string   `json:"level"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success=true")
	}
	if resp.Data.Level != "error" {
		t.Fatalf("expected level=error, got %q", resp.Data.Level)
	}
	if resp.Data.Count != 1 || len(resp.Data.Lines) != 1 {
		t.Fatalf("unexpected count/lines: count=%d lines=%#v", resp.Data.Count, resp.Data.Lines)
	}
	if !strings.Contains(strings.ToLower(resp.Data.Lines[0]), "error") {
		t.Fatalf("expected error line, got %q", resp.Data.Lines[0])
	}
}

func TestHandleLogsExport_ReturnsTextAttachment(t *testing.T) {
	restore := server.SetProcessLogsForTest([]string{"line-a", "line-b"}, 10)
	t.Cleanup(restore)

	h := server.NewHttpServer(&config.RouterServerConfig{}, nil)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/logs/export?limit=2", nil)
	h.HandleLogsExportForTest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Content-Type"), "text/plain") {
		t.Fatalf("unexpected content-type: %s", rr.Header().Get("Content-Type"))
	}
	if !strings.Contains(strings.ToLower(rr.Header().Get("Content-Disposition")), "attachment") {
		t.Fatalf("unexpected content-disposition: %s", rr.Header().Get("Content-Disposition"))
	}
	body := rr.Body.String()
	if !strings.Contains(body, "line-a") || !strings.Contains(body, "line-b") {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestHandleRouterRPCRanking_ReturnsList(t *testing.T) {
	s := server.NewServer(&config.RouterServerConfig{RouterServerPort: 1, HTTPMonitorPort: 2})
	s.RecordRouterRPCForTest("router-a", "router-b")
	s.RecordRouterRPCForTest("router-a", "router-b")

	h := server.NewHttpServer(&config.RouterServerConfig{RouterServerPort: 1, HTTPMonitorPort: 2}, s)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/rpc/router-ranking?limit=5&keyword=router", nil)
	h.HandleRouterRPCRankingForTest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Total int `json:"total"`
			List  []struct {
				RouterID  string `json:"routerId"`
				PerMinute int    `json:"perMinute"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response error: %v", err)
	}
	if !resp.Success || resp.Data.Total == 0 || len(resp.Data.List) == 0 {
		t.Fatalf("unexpected response: %s", rr.Body.String())
	}
}

func TestMatchLogLevel(t *testing.T) {
	if !server.MatchLogLevelForTest("2026 info ready", "info") {
		t.Fatalf("expected info matched")
	}
	if server.MatchLogLevelForTest("2026 warn timeout", "info") {
		t.Fatalf("warn should not match info")
	}
	if !server.MatchLogLevelForTest("2026 error failed", "error") {
		t.Fatalf("error should match")
	}
	if !server.MatchLogLevelForTest("2026 fatal panic", "error") {
		t.Fatalf("fatal should match error")
	}
	if !server.MatchLogLevelForTest("2026 warn timeout", "warn") {
		t.Fatalf("warn should match")
	}
}
