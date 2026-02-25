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
	if indexResp.Code != http.StatusFound {
		t.Fatalf("expected status 302 for / without token, got %d", indexResp.Code)
	}
	if loc := indexResp.Header().Get("Location"); loc != "/login.html" {
		t.Fatalf("expected redirect to /login.html, got %q", loc)
	}

	token, err := GenerateToken("admin")
	if err != nil {
		t.Fatalf("generate token error: %v", err)
	}
	authReq := httptest.NewRequest(http.MethodGet, "/", nil)
	authReq.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
	authResp := httptest.NewRecorder()
	h.ServeHTTP(authResp, authReq)
	if authResp.Code != http.StatusOK {
		t.Fatalf("expected status 200 for / with token, got %d", authResp.Code)
	}
	indexBody, _ := io.ReadAll(authResp.Body)
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

	loginReq := httptest.NewRequest(http.MethodGet, "/login.html", nil)
	loginResp := httptest.NewRecorder()
	h.ServeHTTP(loginResp, loginReq)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /login.html, got %d", loginResp.Code)
	}
}
