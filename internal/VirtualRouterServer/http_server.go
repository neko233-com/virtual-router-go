package VirtualRouterServer

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/neko233-com/virtual-router-go/internal/config"
	"github.com/neko233-com/virtual-router-go/internal/core"
)

type HttpServer struct {
	cfg  *config.RouterServerConfig
	srv  *Server
	http *http.Server
}

func NewHttpServer(cfg *config.RouterServerConfig, srv *Server) *HttpServer {
	return &HttpServer{cfg: cfg, srv: srv}
}

func (h *HttpServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/auth/login", h.handleLogin)
	mux.HandleFunc("/api/auth/refresh", h.handleRefresh)
	mux.HandleFunc("/api/auth/validate", h.handleValidate)
	mux.HandleFunc("/api/auth/logout", h.handleLogout)

	mux.HandleFunc("/api/status", h.withAuth(h.handleStatus))
	mux.HandleFunc("/api/metrics", h.withAuth(h.handleMetrics))
	mux.HandleFunc("/api/routers", h.withAuth(h.handleRouters))
	mux.HandleFunc("/api/connections", h.withAuth(h.handleConnections))
	mux.HandleFunc("/api/rpc-stats", h.withAuth(h.handleRpcStats))
	mux.HandleFunc("/api/rpc/router-ranking", h.withAuth(h.handleRouterRPCRanking))
	mux.HandleFunc("/api/message-stats", h.withAuth(h.handleMessageStats))
	mux.HandleFunc("/api/monitor-stats", h.withAuth(h.handleMonitorStats))
	mux.HandleFunc("/api/viewers", h.withAuth(h.handleViewers))
	mux.HandleFunc("/api/logs", h.withAuth(h.handleLogs))
	mux.HandleFunc("/api/logs/export", h.withAuth(h.handleLogsExport))
	mux.HandleFunc("/api/system/settings", h.withAuth(h.handleSystemSettings))
	mux.HandleFunc("/api/system/admin-password", h.withAuth(h.handleUpdateAdminPassword))

	mux.HandleFunc("/api/debug/validate-route-id", h.withAuth(h.handleValidateRouteId))
	mux.HandleFunc("/api/debug/available-routes", h.withAuth(h.handleAvailableRoutes))
	mux.HandleFunc("/api/debug/send-rpc", h.withAuth(h.handleDebugSendRpc))
	mux.HandleFunc("/api/debug/rpc-result", h.withAuth(h.handleDebugRpcResult))
	mux.HandleFunc("/api/debug/rpc-stubs", h.withAuth(h.handleDebugRpcStubs))
	mux.Handle("/", monitorStaticHandler())

	h.http = &http.Server{
		Addr:              ":" + intToString(h.cfg.HTTPMonitorPort),
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		_ = h.http.Shutdown(context.Background())
	}()

	slog.Info("HTTP Monitor 启动成功", "port", h.cfg.HTTPMonitorPort)
	logHTTPAccessURLs("HTTP Monitor", h.cfg.HTTPMonitorPort)
	return h.http.ListenAndServe()
}

func (h *HttpServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"success": false, "message": "method not allowed"})
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if h.cfg.AdminPassword == "" {
		writeJSON(w, http.StatusForbidden, map[string]any{"success": false, "message": "服务端未配置 adminPassword，禁止登录"})
		return
	}
	if req.Password == h.cfg.AdminPassword {
		token, _ := GenerateToken("admin")
		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "登录成功",
			"data": map[string]any{
				"token":     token,
				"expiresIn": 24 * 60 * 60,
				"tokenType": "Bearer",
				"user": map[string]any{
					"id":   "admin",
					"name": "Administrator",
					"role": "admin",
				},
			},
		})
		return
	}
	writeJSON(w, http.StatusUnauthorized, map[string]any{"success": false, "message": "密码错误"})
}

func (h *HttpServer) handleRefresh(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"success": false, "message": "缺少 Token"})
		return
	}
	newToken, ok := RefreshToken(token)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"success": false, "message": "Token 无效，请重新登录"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Token 刷新成功",
		"data": map[string]any{
			"token":     newToken,
			"expiresIn": 24 * 60 * 60,
			"tokenType": "Bearer",
		},
	})
}

func (h *HttpServer) handleValidate(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"success": false, "valid": false, "message": "缺少 Token"})
		return
	}
	valid := ValidateToken(token)
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"valid":   valid,
		"data": map[string]any{
			"remainingSeconds": GetTokenRemainingSeconds(token),
			"shouldRefresh":    ShouldRefreshToken(token),
		},
	})
}

func (h *HttpServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "message": "登出成功"})
}

func (h *HttpServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	_, _, _, _, uptime := h.srv.Stats()
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"serverInfo": map[string]any{
				"name":      "Virtual Router Center",
				"version":   "0.0.10",
				"uptime":    uptime,
				"startTime": time.Now().Add(-time.Duration(uptime) * time.Millisecond).UnixMilli(),
			},
			"system": map[string]any{
				"osName":      runtime.GOOS,
				"osVersion":   "-",
				"javaVersion": "-",
				"processors":  runtime.NumCPU(),
			},
			"router": map[string]any{
				"port":        h.cfg.RouterServerPort,
				"monitorPort": h.cfg.HTTPMonitorPort,
			},
		},
	})
}

func (h *HttpServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	used := int64(m.Alloc)
	max := int64(m.Sys)
	usagePercent := 0
	if max > 0 {
		usagePercent = int((used * 100) / max)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"cpu": map[string]any{
				"usage":       0,
				"loadAverage": 0,
				"cores":       runtime.NumCPU(),
			},
			"memory": map[string]any{
				"used":         used,
				"max":          max,
				"total":        max,
				"free":         max - used,
				"usagePercent": usagePercent,
			},
			"thread": map[string]any{
				"count": runtime.NumGoroutine(),
				"peak":  runtime.NumGoroutine(),
			},
			"gc": []any{
				map[string]any{
					"name":            "GC",
					"collectionCount": m.NumGC,
					"collectionTime":  int64(m.PauseTotalNs) / int64(time.Millisecond),
				},
			},
		},
	})
}

func (h *HttpServer) handleRouters(w http.ResponseWriter, r *http.Request) {
	nodes := h.srv.SessionManager().GetAllSessionSnapshots()
	keyword := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("keyword")))
	routers := make([]any, 0, len(nodes))
	for _, n := range nodes {
		if keyword != "" && !strings.Contains(strings.ToLower(n.RouterId), keyword) {
			continue
		}
		isRelay := n.HostForRpc == "" || n.PortForRpc == 0
		rpcMode := "direct"
		address := n.HostForRpc + ":" + intToString(n.PortForRpc)
		if isRelay {
			rpcMode = "relay"
			address = "-"
		}
		geo := resolveIPGeo(n.RemoteIP)
		location := strings.TrimSpace(strings.TrimSpace(geo.RegionName) + " " + strings.TrimSpace(geo.City))
		if location == "" {
			location = "-"
		}
		countryLabel := geo.Country
		if geo.CountryCode != "" && geo.CountryCode != "LOCAL" && geo.CountryCode != "LAN" {
			countryLabel = geo.Country + " (" + geo.CountryCode + ")"
		}
		if countryLabel == "" {
			countryLabel = "未知"
		}

		routers = append(routers, map[string]any{
			"routeId":       n.RouterId,
			"rpcHost":       n.HostForRpc,
			"rpcPort":       n.PortForRpc,
			"address":       address,
			"remoteAddr":    n.RemoteAddr,
			"remoteIp":      n.RemoteIP,
			"remotePort":    n.RemotePort,
			"country":       countryLabel,
			"countryCode":   geo.CountryCode,
			"region":        geo.RegionName,
			"city":          geo.City,
			"location":      location,
			"isp":           geo.ISP,
			"org":           geo.Org,
			"as":            geo.AS,
			"stubCount":     n.StubCount,
			"rpcMode":       rpcMode,
			"status":        "ONLINE",
			"connected":     true,
			"lastHeartbeat": n.LastHeartbeatMs,
			"uptime":        0,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"routers": routers,
			"list":    routers,
			"total":   len(routers),
			"online":  len(routers),
			"offline": 0,
		},
	})
}

func (h *HttpServer) handleRouterRPCRanking(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if value := strings.TrimSpace(r.URL.Query().Get("limit")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			if parsed > 500 {
				parsed = 500
			}
			limit = parsed
		}
	}
	keyword := strings.TrimSpace(r.URL.Query().Get("keyword"))
	list := h.srv.RouterRPCStats(keyword, limit)

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"list":    list,
			"total":   len(list),
			"keyword": keyword,
			"limit":   limit,
		},
	})
}

func (h *HttpServer) handleConnections(w http.ResponseWriter, r *http.Request) {
	totalConn, _, _, totalRequests, _ := h.srv.Stats()
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"connectionHistory": []any{},
			"totalConnections":  totalConn,
			"totalRequests":     totalRequests,
		},
	})
}

func (h *HttpServer) handleRpcStats(w http.ResponseWriter, r *http.Request) {
	_, _, totalBytes, totalRequests, _ := h.srv.Stats()
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"load": map[string]any{
				"percent": 0,
				"color":   "#00ff41",
				"status":  "正常",
			},
			"total": map[string]any{
				"messages": totalRequests,
				"bytes":    totalBytes,
				"errors":   0,
			},
		},
	})
}

func (h *HttpServer) handleMessageStats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"messageTypes": []any{},
		},
	})
}

func (h *HttpServer) handleMonitorStats(w http.ResponseWriter, r *http.Request) {
	_, currentConn, _, totalRequests, _ := h.srv.Stats()
	requestsLastMinute := h.srv.RequestsPerMinute()
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"totalRequests":      totalRequests,
			"requestsLastMinute": requestsLastMinute,
			"activeViewers":      currentConn,
			"uptime":             0,
			"totalConnections":   currentConn,
		},
	})
}

func (h *HttpServer) handleViewers(w http.ResponseWriter, r *http.Request) {
	_, currentConn, _, _, _ := h.srv.Stats()
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"activeViewers": currentConn,
		},
	})
}

func (h *HttpServer) handleLogs(w http.ResponseWriter, r *http.Request) {
	limit := 200
	keyword := strings.TrimSpace(r.URL.Query().Get("keyword"))
	level := normalizeLogLevel(r.URL.Query().Get("level"))
	if value := r.URL.Query().Get("limit"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			if parsed > 1000 {
				parsed = 1000
			}
			limit = parsed
		}
	}

	lines := GetRecentProcessLogs(1000)
	if keyword != "" {
		filtered := make([]string, 0, len(lines))
		lowerKeyword := strings.ToLower(keyword)
		for _, line := range lines {
			if strings.Contains(strings.ToLower(line), lowerKeyword) {
				filtered = append(filtered, line)
			}
		}
		lines = filtered
	}
	if level != "all" {
		filtered := make([]string, 0, len(lines))
		for _, line := range lines {
			if matchLogLevel(line, level) {
				filtered = append(filtered, line)
			}
		}
		lines = filtered
	}
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"lines":   lines,
			"count":   len(lines),
			"keyword": keyword,
			"level":   level,
		},
	})
}

func (h *HttpServer) handleLogsExport(w http.ResponseWriter, r *http.Request) {
	limit := 1000
	keyword := strings.TrimSpace(r.URL.Query().Get("keyword"))
	level := normalizeLogLevel(r.URL.Query().Get("level"))
	if value := r.URL.Query().Get("limit"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			if parsed > 5000 {
				parsed = 5000
			}
			limit = parsed
		}
	}

	lines := GetRecentProcessLogs(limit)
	if keyword != "" {
		filtered := make([]string, 0, len(lines))
		lowerKeyword := strings.ToLower(keyword)
		for _, line := range lines {
			if strings.Contains(strings.ToLower(line), lowerKeyword) {
				filtered = append(filtered, line)
			}
		}
		lines = filtered
	}
	if level != "all" {
		filtered := make([]string, 0, len(lines))
		for _, line := range lines {
			if matchLogLevel(line, level) {
				filtered = append(filtered, line)
			}
		}
		lines = filtered
	}

	fileName := "router-logs-" + time.Now().Format("20060102-150405") + ".txt"
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+fileName+"\"")
	_, _ = w.Write([]byte(strings.Join(lines, "\n")))
}

func normalizeLogLevel(raw string) string {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "info", "warn", "error", "all":
		return v
	default:
		return "all"
	}
}

func matchLogLevel(line, level string) bool {
	lower := strings.ToLower(line)
	switch level {
	case "error":
		return strings.Contains(lower, "error") || strings.Contains(lower, "fatal")
	case "warn":
		return strings.Contains(lower, "warn")
	case "info":
		if strings.Contains(lower, "error") || strings.Contains(lower, "fatal") || strings.Contains(lower, "warn") {
			return false
		}
		return true
	default:
		return true
	}
}

func (h *HttpServer) handleSystemSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"routerServerPort":        h.cfg.RouterServerPort,
			"httpMonitorPort":         h.cfg.HTTPMonitorPort,
			"adminPasswordConfigured": strings.TrimSpace(h.cfg.AdminPassword) != "",
			"logBufferCapacity":       800,
		},
	})
}

func (h *HttpServer) handleUpdateAdminPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"success": false, "message": "method not allowed"})
		return
	}

	var req struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": "请求格式错误"})
		return
	}
	if req.OldPassword != h.cfg.AdminPassword {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": "旧密码不正确"})
		return
	}
	if len(strings.TrimSpace(req.NewPassword)) < 4 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": "新密码长度至少 4 位"})
		return
	}

	h.cfg.AdminPassword = strings.TrimSpace(req.NewPassword)
	if err := config.WriteRouterServerConfig("", h.cfg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "message": "更新成功但写入配置失败: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "管理员密码已更新并写入配置文件",
	})
}

func (h *HttpServer) handleValidateRouteId(w http.ResponseWriter, r *http.Request) {
	routeId := r.URL.Query().Get("routeId")
	if strings.TrimSpace(routeId) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": "路由 ID 不能为空"})
		return
	}
	session := h.srv.SessionManager().GetSession(routeId)
	exists := session != nil
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"exists":  exists,
		"routeId": routeId,
		"message": map[bool]string{true: "路由节点存在", false: "路由节点不存在"}[exists],
	})
}

func (h *HttpServer) handleAvailableRoutes(w http.ResponseWriter, r *http.Request) {
	nodes := h.srv.SessionManager().GetAllRouteNodeList()
	ids := make([]string, 0, len(nodes))
	for _, n := range nodes {
		ids = append(ids, n.RouterId)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"routeIds": ids,
			"count":    len(ids),
		},
	})
}

func (h *HttpServer) handleDebugSendRpc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"success": false, "message": "method not allowed"})
		return
	}
	var req struct {
		TargetRouteId string `json:"targetRouteId"`
		PacketId      int    `json:"packetId"`
		Params        []any  `json:"params"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.TargetRouteId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": "路由节点不存在"})
		return
	}
	session := h.srv.SessionManager().GetSession(req.TargetRouteId)
	if session == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "message": "路由节点不存在: " + req.TargetRouteId})
		return
	}
	if req.PacketId <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": "Packet ID 必须大于 0"})
		return
	}

	stub := findStub(session.RpcServerInfo.Stubs, req.PacketId)
	if stub == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": "目标节点未注册 Packet ID = " + intToString(req.PacketId)})
		return
	}

	requestId := intToString(int(time.Now().UnixMilli()))
	argsJson := make([]string, 0, len(req.Params))
	for _, p := range req.Params {
		b, _ := json.Marshal(p)
		argsJson = append(argsJson, string(b))
	}

	rpcReq := map[string]any{
		"rpcUid":             requestId,
		"packetId":           req.PacketId,
		"methodArgsJsonList": argsJson,
		"fromDebug":          true,
	}
	dataBytes, _ := json.Marshal(rpcReq)
	dataStr := string(dataBytes)
	mt := core.RouteMessageTypeRpcRequest
	msg := &core.RouteMessage{
		FromRouteId: "debug-admin",
		ToRouteId:   req.TargetRouteId,
		MessageType: &mt,
		Data:        &dataStr,
	}
	_ = session.WriteRouteMessage(msg)

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "✅ RPC 调试请求已发送",
		"data": map[string]any{
			"targetRouteId": req.TargetRouteId,
			"packetId":      req.PacketId,
			"method":        stub.ClassName + "." + stub.MethodName,
			"paramsCount":   len(req.Params),
			"requestId":     requestId,
			"status":        "sent",
			"note":          "RPC 请求已通过 Router Center 转发到目标节点，等待响应（异步调用）",
		},
	})
}

func (h *HttpServer) handleDebugRpcResult(w http.ResponseWriter, r *http.Request) {
	requestId := r.URL.Query().Get("requestId")
	if requestId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": "requestId 不能为空"})
		return
	}
	if result, ok := h.srv.GetDebugResult(requestId); ok {
		var obj any
		if err := json.Unmarshal([]byte(result), &obj); err != nil {
			obj = result
		}
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": obj})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": false,
		"message": "结果尚未就绪或已过期",
		"data": map[string]any{
			"requestId": requestId,
			"status":    "pending",
		},
	})
}

func (h *HttpServer) handleDebugRpcStubs(w http.ResponseWriter, r *http.Request) {
	routeId := r.URL.Query().Get("routeId")
	if strings.TrimSpace(routeId) == "" {
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": []any{}})
		return
	}
	session := h.srv.SessionManager().GetSession(routeId)
	if session == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "message": "路由节点不存在"})
		return
	}
	stubs := make([]any, 0, len(session.RpcServerInfo.Stubs))
	for _, s := range session.RpcServerInfo.Stubs {
		desc := s.Description
		if desc == "" {
			desc = "[" + intToString(s.PacketId) + "] " + afterLast(s.ClassName, ".") + "." + s.MethodName
		}
		stubs = append(stubs, map[string]any{
			"packetId":              s.PacketId,
			"className":             s.ClassName,
			"methodName":            s.MethodName,
			"parameterTypes":        s.ParameterTypes,
			"parameterNames":        s.ParameterNames,
			"parameterDescriptions": s.ParameterDescriptions,
			"parameterExampleJson":  s.ParameterExampleJson,
			"description":           desc,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": stubs})
}

func (h *HttpServer) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/auth/") {
			next(w, r)
			return
		}
		if r.Method == http.MethodOptions {
			next(w, r)
			return
		}
		token := extractToken(r)
		if token == "" || !ValidateToken(token) {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"success": false, "message": "未授权，请先登录"})
			return
		}
		next(w, r)
	}
}

func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, obj any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(obj)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Expose-Headers", "Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func findStub(stubs []core.RpcStubMetadata, packetId int) *core.RpcStubMetadata {
	for i := range stubs {
		if stubs[i].PacketId == packetId {
			return &stubs[i]
		}
	}
	return nil
}

func afterLast(s, sep string) string {
	idx := strings.LastIndex(s, sep)
	if idx < 0 {
		return s
	}
	return s[idx+len(sep):]
}
