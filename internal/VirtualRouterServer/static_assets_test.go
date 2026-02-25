package VirtualRouterServer

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMonitorStaticHandler_ServesIndexAndAppJS(t *testing.T) {
	h := monitorStaticHandler()

	indexReq := httptest.NewRequest(http.MethodGet, "/", nil)
	indexResp := httptest.NewRecorder()
	h.ServeHTTP(indexResp, indexReq)
	if indexResp.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /, got %d", indexResp.Code)
	}
	indexBody, _ := io.ReadAll(indexResp.Body)
	if !strings.Contains(string(indexBody), "Virtual Router Monitor") {
		t.Fatalf("index html does not contain expected title")
	}

	appReq := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	appResp := httptest.NewRecorder()
	h.ServeHTTP(appResp, appReq)
	if appResp.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /app.js, got %d", appResp.Code)
	}
	appBody, _ := io.ReadAll(appResp.Body)
	if !strings.Contains(string(appBody), "const state") {
		t.Fatalf("app.js does not contain expected content")
	}
}
