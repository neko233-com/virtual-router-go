package VirtualRouterServer

import (
	"io"
	"net/http"

	"github.com/neko233-com/virtual-router-go/internal/core"
)

// 以下导出方法仅用于跨目录单元测试，避免测试直接依赖包内私有符号。

func (s *Server) ForwardToTargetForTest(msg *core.RouteMessage) {
	s.forwardToTarget(msg)
}

func (s *Server) RecordRouterRPCForTest(fromRouteID, toRouteID string) {
	s.recordRouterRPC(fromRouteID, toRouteID)
}

func (s *Server) SetRouterLastMinuteHitsForTest(routerID string, hits []int64) {
	s.rpcStatsMu.Lock()
	defer s.rpcStatsMu.Unlock()
	st, ok := s.rpcStatsByRouter[routerID]
	if !ok {
		st = s.ensureRouterRPCStats(routerID)
	}
	st.LastMinuteHits = hits
}

func (h *HttpServer) HandleLogsForTest(w http.ResponseWriter, r *http.Request) {
	h.handleLogs(w, r)
}

func (h *HttpServer) HandleLogsExportForTest(w http.ResponseWriter, r *http.Request) {
	h.handleLogsExport(w, r)
}

func (h *HttpServer) HandleRouterRPCRankingForTest(w http.ResponseWriter, r *http.Request) {
	h.handleRouterRPCRanking(w, r)
}

func (h *HttpServer) HandleUpdateAdminPasswordForTest(w http.ResponseWriter, r *http.Request) {
	h.handleUpdateAdminPassword(w, r)
}

func MatchLogLevelForTest(line, level string) bool {
	return matchLogLevel(line, level)
}

func ResolveIPCountryForTest(ip string) string {
	return resolveIPCountry(ip)
}

func MonitorStaticHandlerForTest() http.Handler {
	return monitorStaticHandler()
}

func AuthCookieNameForTest() string {
	return authCookieName
}

func SetProcessLogsForTest(lines []string, capacity int) (restore func()) {
	globalLogs.mu.Lock()
	backupLines := append([]string(nil), globalLogs.lines...)
	backupCapacity := globalLogs.capacity
	globalLogs.lines = append([]string(nil), lines...)
	if capacity > 0 {
		globalLogs.capacity = capacity
	}
	globalLogs.mu.Unlock()

	return func() {
		globalLogs.mu.Lock()
		globalLogs.lines = backupLines
		globalLogs.capacity = backupCapacity
		globalLogs.mu.Unlock()
	}
}

type LogCaptureTestHelper struct {
	capture *logCapture
}

func NewLogCaptureTestHelper(capacity int) *LogCaptureTestHelper {
	if capacity <= 0 {
		capacity = 1
	}
	return &LogCaptureTestHelper{capture: &logCapture{capacity: capacity, lines: make([]string, 0, capacity)}}
}

func (h *LogCaptureTestHelper) Write(p []byte) (int, error) {
	return h.capture.Write(p)
}

func (h *LogCaptureTestHelper) GetRecent(limit int) []string {
	return h.capture.getRecent(limit)
}

func NewRotatingFileWriterForTest(dir, baseName string, maxBytes int64, maxFiles int) (io.WriteCloser, error) {
	return newRotatingFileWriter(logRotationConfig{Dir: dir, BaseName: baseName, MaxBytes: maxBytes, MaxFiles: maxFiles})
}
