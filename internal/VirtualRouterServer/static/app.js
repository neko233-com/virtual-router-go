const state = {
  token: "",
  activeTab: "home",
  stubs: [],
  history: {
    labels: [],
    totalRequests: [],
    totalConnections: [],
    onlineRouters: [],
    memoryUsagePercent: [],
  },
  charts: {
    request: null,
    memory: null,
    online: null,
    rpcRank: null,
    memoryUsageTrend: null,
    countryDist: null,
    modeDist: null,
    stubTop: null,
  },
};

const AUTH_TOKEN_KEY = "virtual-router-admin-token";

const refreshBtn = document.getElementById("refreshBtn");
const logoutBtn = document.getElementById("logoutBtn");
const appMsg = document.getElementById("appMsg");
const tabs = document.getElementById("tabs");

const stats = document.getElementById("stats");
const routersBody = document.getElementById("routersBody");
const rpcRankBody = document.getElementById("rpcRankBody");
const settingsInfo = document.getElementById("settingsInfo");
const routerKeywordInput = document.getElementById("routerKeyword");
const searchRoutersBtn = document.getElementById("searchRoutersBtn");
const rpcTrafficKeywordInput = document.getElementById("rpcTrafficKeyword");
const searchRpcTrafficBtn = document.getElementById("searchRpcTrafficBtn");
const logLimitInput = document.getElementById("logLimit");
const logKeywordInput = document.getElementById("logKeyword");
const logLevelInput = document.getElementById("logLevel");
const logAutoRefreshInput = document.getElementById("logAutoRefresh");
const loadLogsBtn = document.getElementById("loadLogsBtn");
const exportLogsBtn = document.getElementById("exportLogsBtn");
const logOutput = document.getElementById("logOutput");
const oldPasswordInput = document.getElementById("oldPassword");
const newPasswordInput = document.getElementById("newPassword");
const confirmPasswordInput = document.getElementById("confirmPassword");
const updatePasswordBtn = document.getElementById("updatePasswordBtn");
const settingsMsg = document.getElementById("settingsMsg");

const targetRouteIdInput = document.getElementById("targetRouteId");
const packetIdInput = document.getElementById("packetId");
const loadStubsBtn = document.getElementById("loadStubsBtn");
const stubSelect = document.getElementById("stubSelect");
const paramText = document.getElementById("paramText");
const sendRpcBtn = document.getElementById("sendRpcBtn");
const rpcMsg = document.getElementById("rpcMsg");
const rpcResult = document.getElementById("rpcResult");
const openRpcDebugBtn = document.getElementById("openRpcDebugBtn");
const closeRpcDebugBtn = document.getElementById("closeRpcDebugBtn");
const rpcDebugBackdrop = document.getElementById("rpcDebugBackdrop");
const rpcDebugModal = document.getElementById("rpcDebugModal");

refreshBtn.addEventListener("click", () => loadAll());
logoutBtn.addEventListener("click", logout);
tabs.addEventListener("click", onTabClick);
loadLogsBtn.addEventListener("click", () => loadLogs());
exportLogsBtn.addEventListener("click", exportLogs);
searchRoutersBtn.addEventListener("click", () => loadRoutersAndRanking());
searchRpcTrafficBtn.addEventListener("click", () => loadRoutersAndRanking());

loadStubsBtn.addEventListener("click", loadStubs);
sendRpcBtn.addEventListener("click", sendRpc);
stubSelect.addEventListener("change", onStubChange);
updatePasswordBtn.addEventListener("click", updateAdminPassword);
openRpcDebugBtn.addEventListener("click", openRpcDebugModal);
closeRpcDebugBtn.addEventListener("click", closeRpcDebugModal);
rpcDebugBackdrop.addEventListener("click", closeRpcDebugModal);

window.addEventListener("resize", resizeCharts);

initCharts();
restoreAuthState();

function enableAuthorizedUI() {
  refreshBtn.disabled = false;
  logoutBtn.disabled = false;
  loadStubsBtn.disabled = false;
  sendRpcBtn.disabled = false;
  stubSelect.disabled = false;
  loadLogsBtn.disabled = false;
  exportLogsBtn.disabled = false;
  updatePasswordBtn.disabled = false;
  searchRoutersBtn.disabled = false;
  searchRpcTrafficBtn.disabled = false;
  openRpcDebugBtn.disabled = false;
}

async function restoreAuthState() {
  const token = localStorage.getItem(AUTH_TOKEN_KEY);
  let ok = false;
  if (token) {
    ok = await validateOrRefreshToken(token);
  } else {
    ok = await refreshTokenFromServerSession();
  }
  if (!ok) {
    localStorage.removeItem(AUTH_TOKEN_KEY);
    state.token = "";
    redirectToLogin("登录已过期，请重新登录");
    return;
  }

  enableAuthorizedUI();
  setMessage("已恢复登录状态");
  await loadAll(true);
  startAutoRefresh();
}

async function refreshTokenFromServerSession() {
  try {
    const refreshResp = await fetch("/api/auth/refresh");
    if (!refreshResp.ok) {
      return false;
    }
    const refreshData = await refreshResp.json();
    const newToken = refreshData?.data?.token || "";
    if (!newToken) {
      return false;
    }
    state.token = newToken;
    localStorage.setItem(AUTH_TOKEN_KEY, newToken);
    return true;
  } catch (_) {
    return false;
  }
}

async function validateOrRefreshToken(token) {
  try {
    const validateResp = await fetch("/api/auth/validate", {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (validateResp.ok) {
      const validateData = await validateResp.json();
      if (validateData.valid === true) {
        state.token = token;
        return true;
      }
    }
  } catch (_) {
  }

  try {
    const refreshResp = await fetch("/api/auth/refresh", {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (!refreshResp.ok) {
      return false;
    }
    const refreshData = await refreshResp.json();
    const newToken = refreshData?.data?.token || "";
    if (!newToken) {
      return false;
    }
    state.token = newToken;
    localStorage.setItem(AUTH_TOKEN_KEY, newToken);
    return true;
  } catch (_) {
    return false;
  }
}

function startAutoRefresh() {
  if (state._autoTimer) {
    return;
  }
  state._autoTimer = setInterval(async () => {
    if (!state.token) {
      return;
    }
    await loadAll(true);
    if (state.activeTab === "logs" && logAutoRefreshInput.checked) {
      await loadLogs();
    }
  }, 5000);
}

function onTabClick(event) {
  const target = event.target;
  if (!target.classList.contains("tab")) {
    return;
  }
  const tab = target.dataset.tab;
  if (!tab) {
    return;
  }

  state.activeTab = tab;

  document.querySelectorAll(".tab").forEach((item) => {
    item.classList.toggle("active", item.dataset.tab === tab);
  });
  document.querySelectorAll(".panel").forEach((item) => {
    item.classList.toggle("active", item.id === `tab-${tab}`);
  });

  if (tab === "logs" && state.token) {
    loadLogs();
  }
  if (tab === "home") {
    resizeCharts();
  }
}

async function loadAll(isSilent = false) {
  if (!state.token) {
    setMessage("请先登录");
    return;
  }
  if (!isSilent) {
    setMessage("加载中...");
  }

  try {
    const [statusRes, metricsRes, connRes, routersRes, monitorRes, viewersRes, rpcStatsRes, rpcRankRes, systemSettingsRes] = await Promise.all([
      apiGet("/api/status"),
      apiGet("/api/metrics"),
      apiGet("/api/connections"),
      apiGet(`/api/routers?keyword=${encodeURIComponent((routerKeywordInput.value || "").trim())}`),
      apiGet("/api/monitor-stats"),
      apiGet("/api/viewers"),
      apiGet("/api/rpc-stats"),
      apiGet(`/api/rpc/router-ranking?limit=50&keyword=${encodeURIComponent((rpcTrafficKeywordInput.value || "").trim())}`),
      apiGet("/api/system/settings"),
    ]);

    const serverInfo = statusRes.data?.serverInfo || {};
    const routerInfo = statusRes.data?.router || {};
    const systemInfo = statusRes.data?.system || {};
    const memory = metricsRes.data?.memory || {};
    const cpu = metricsRes.data?.cpu || {};
    const connData = connRes.data || {};
    const monitor = monitorRes.data || {};
    const viewers = viewersRes.data || {};
    const rpcTotal = rpcStatsRes.data?.total || {};
    const rpcRanking = rpcRankRes.data?.list || [];
    const systemSettings = systemSettingsRes.data || {};
    const routersData = routersRes.data || {};
    const routers = routersData.routers || routersData.list || [];

    renderStats([
      ["运行时长", formatDuration(serverInfo.uptime || 0)],
      ["总请求数", connData.totalRequests || 0],
      ["历史连接数", connData.totalConnections || 0],
      ["在线路由", routersData.online || routers.length || 0],
      ["活跃查看者", viewers.activeViewers || 0],
      ["累计流量", formatBytesToMB(rpcTotal.bytes || 0)],
    ]);
    renderRouters(routers);
    renderRPCRankTable(rpcRanking);
    renderRPCRankChart(rpcRanking);
    renderCountryDistributionChart(routers);
    renderModeDistributionChart(routers);
    renderStubTopChart(routers);
    syncRouteOptions(routers);
    renderSettings({
      osName: systemInfo.osName || "-",
      cpuCores: systemInfo.processors || 0,
      routerPort: routerInfo.port || 0,
      monitorPort: routerInfo.monitorPort || 0,
      cpuUsage: cpu.usage || 0,
      memoryUsed: formatBytesToMB(memory.used || 0),
      memoryMax: formatBytesToMB(memory.max || 0),
      requestsLastMinute: monitor.requestsLastMinute || 0,
      adminPasswordConfigured: systemSettings.adminPasswordConfigured ? "是" : "否",
      logBufferCapacity: systemSettings.logBufferCapacity || 0,
    });

    updateHistory({
      totalRequests: monitor.requestsLastMinute || 0,
      totalConnections: connData.totalConnections || 0,
      onlineRouters: routersData.online || routers.length || 0,
      memoryUsed: memory.used || 0,
      memoryMax: memory.max || 0,
      memoryUsagePercent: memory.usagePercent || 0,
    });
    refreshCharts();

    if (!isSilent) {
      setMessage(`刷新成功，节点数: ${routers.length}`);
    }
  } catch (error) {
    setMessage(`加载失败: ${error.message || error}`);
  }
}

async function loadRoutersAndRanking() {
  if (!state.token) {
    return;
  }
  await loadAll(true);
}

async function loadLogs() {
  if (!state.token) {
    return;
  }
  const limit = Math.min(1000, Math.max(10, Number(logLimitInput.value || 200)));
  const keyword = (logKeywordInput.value || "").trim();
  const level = (logLevelInput.value || "all").trim();
  try {
    const data = await apiGet(`/api/logs?limit=${limit}&keyword=${encodeURIComponent(keyword)}&level=${encodeURIComponent(level)}`);
    const lines = data?.data?.lines || [];
    logOutput.textContent = lines.length ? lines.join("\n") : "暂无日志";
  } catch (error) {
    logOutput.textContent = `日志加载失败: ${error.message || error}`;
  }
}

async function exportLogs() {
  if (!state.token) {
    return;
  }
  const limit = Math.min(5000, Math.max(10, Number(logLimitInput.value || 200)));
  const keyword = (logKeywordInput.value || "").trim();
  const level = (logLevelInput.value || "all").trim();
  const url = `/api/logs/export?limit=${limit}&keyword=${encodeURIComponent(keyword)}&level=${encodeURIComponent(level)}`;

  try {
    const resp = await fetch(url, {
      headers: { Authorization: `Bearer ${state.token}` },
    });
    if (!resp.ok) {
      throw new Error(`导出失败: HTTP ${resp.status}`);
    }

    const blob = await resp.blob();
    const fileName = parseFileName(resp.headers.get("Content-Disposition")) || "router-logs.txt";
    const objectUrl = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = objectUrl;
    link.download = fileName;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(objectUrl);
  } catch (error) {
    logOutput.textContent = `导出失败: ${error.message || error}`;
  }
}

function parseFileName(contentDisposition) {
  if (!contentDisposition) {
    return "";
  }
  const marker = "filename=";
  const idx = contentDisposition.toLowerCase().indexOf(marker);
  if (idx < 0) {
    return "";
  }
  return contentDisposition.substring(idx + marker.length).trim().replaceAll('"', "");
}

async function apiGet(path) {
  const resp = await fetch(path, {
    headers: { Authorization: `Bearer ${state.token}` },
  });
  const data = await resp.json();
  if (!resp.ok || data.success === false) {
    throw new Error(data.message || `请求失败: ${path}`);
  }
  return data;
}

async function logout() {
  try {
    await fetch("/api/auth/logout", { method: "POST", headers: state.token ? { Authorization: `Bearer ${state.token}` } : undefined });
  } catch (_) {
  }
  localStorage.removeItem(AUTH_TOKEN_KEY);
  state.token = "";
  redirectToLogin();
}

function redirectToLogin(message = "") {
  if (message) {
    const encoded = encodeURIComponent(message);
    window.location.href = `/login.html?msg=${encoded}`;
    return;
  }
  window.location.href = "/login.html";
}

async function apiPost(path, body) {
  const resp = await fetch(path, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${state.token}`,
    },
    body: JSON.stringify(body),
  });
  const data = await resp.json();
  if (!resp.ok || data.success === false) {
    throw new Error(data.message || `请求失败: ${path}`);
  }
  return data;
}

function renderStats(items) {
  stats.innerHTML = items
    .map(([key, value]) => `<div class="stat"><div class="k">${escapeHtml(String(key))}</div><div class="v">${escapeHtml(String(value))}</div></div>`)
    .join("");
}

function renderSettings(info) {
  const pairs = [
    ["操作系统", info.osName],
    ["CPU 核心", info.cpuCores],
    ["Router 端口", info.routerPort],
    ["Monitor 端口", info.monitorPort],
    ["CPU 使用率", `${info.cpuUsage}%`],
    ["内存已用", info.memoryUsed],
    ["内存总量", info.memoryMax],
    ["最近一分钟请求", info.requestsLastMinute],
    ["管理员密码已配置", info.adminPasswordConfigured],
    ["日志缓冲区容量", info.logBufferCapacity],
  ];
  settingsInfo.innerHTML = pairs
    .map(([k, v]) => `<div class="kv-k">${escapeHtml(String(k))}</div><div class="kv-v">${escapeHtml(String(v))}</div>`)
    .join("");
}

function renderRouters(list) {
  if (!list.length) {
    routersBody.innerHTML = `<tr><td colspan="10">暂无在线路由节点</td></tr>`;
    return;
  }
  routersBody.innerHTML = list
    .map((r) => {
      const routeId = r.routeId || "-";
      const lastHeartbeat = formatDateTime(r.lastHeartbeat || 0);
      const remote = [r.remoteIp || "-", r.remotePort || "-"].join(":");
      const country = r.country || "未知";
      const location = r.location || [r.region || "", r.city || ""].filter(Boolean).join(" ") || "-";
      const provider = [r.isp || "-", r.org || "-"].filter(Boolean).join(" / ");
      const mode = r.rpcMode || "-";
      const addr = r.address || "-";
      const stubCount = Number(r.stubCount || 0);
      const status = r.status || (r.connected ? "ONLINE" : "OFFLINE");
      const statusClass = String(status).toUpperCase() === "ONLINE" ? "status-badge status-online" : "status-badge status-offline";
      return `<tr><td>${escapeHtml(routeId)}</td><td>${escapeHtml(lastHeartbeat)}</td><td>${escapeHtml(remote)}</td><td>${escapeHtml(country)}</td><td>${escapeHtml(location)}</td><td>${escapeHtml(provider)}</td><td>${escapeHtml(mode)}</td><td>${escapeHtml(addr)}</td><td>${escapeHtml(String(stubCount))}</td><td><span class="${statusClass}">${escapeHtml(status)}</span></td></tr>`;
    })
    .join("");
}

function renderCountryDistributionChart(routers) {
  if (!state.charts.countryDist) {
    return;
  }
  const bucket = new Map();
  routers.forEach((item) => {
    const key = item.country || "未知";
    bucket.set(key, (bucket.get(key) || 0) + 1);
  });
  const data = Array.from(bucket.entries())
    .map(([name, value]) => ({ name, value }))
    .sort((a, b) => b.value - a.value)
    .slice(0, 12);

  state.charts.countryDist.setOption({
    tooltip: { trigger: "item" },
    series: [
      {
        type: "pie",
        radius: ["38%", "70%"],
        data,
      },
    ],
  });
}

function renderModeDistributionChart(routers) {
  if (!state.charts.modeDist) {
    return;
  }
  const relay = routers.filter((item) => String(item.rpcMode || "").toLowerCase() === "relay").length;
  const direct = routers.filter((item) => String(item.rpcMode || "").toLowerCase() === "direct").length;
  state.charts.modeDist.setOption({
    tooltip: { trigger: "item" },
    series: [
      {
        type: "pie",
        radius: ["40%", "72%"],
        data: [
          { name: "Direct", value: direct },
          { name: "Relay", value: relay },
        ],
      },
    ],
  });
}

function renderStubTopChart(routers) {
  if (!state.charts.stubTop) {
    return;
  }
  const sorted = [...routers]
    .map((item) => ({ routeId: item.routeId || "-", stubCount: Number(item.stubCount || 0) }))
    .sort((a, b) => b.stubCount - a.stubCount)
    .slice(0, 10);

  state.charts.stubTop.setOption({
    tooltip: { trigger: "axis" },
    xAxis: { type: "value" },
    yAxis: { type: "category", data: sorted.map((item) => item.routeId), inverse: true },
    series: [
      {
        type: "bar",
        data: sorted.map((item) => item.stubCount),
      },
    ],
  });
}

function renderRPCRankTable(list) {
  if (!list.length) {
    rpcRankBody.innerHTML = `<tr><td colspan="6">暂无 RPC 统计数据</td></tr>`;
    return;
  }
  rpcRankBody.innerHTML = list
    .map((item, idx) => {
      return `<tr>
        <td>${idx + 1}</td>
        <td>${escapeHtml(item.routerId || "-")}</td>
        <td>${escapeHtml(String(item.perMinute || 0))}</td>
        <td>${escapeHtml(String(item.total || 0))}</td>
        <td>${escapeHtml(String(item.incomingTotal || 0))}</td>
        <td>${escapeHtml(String(item.outgoingTotal || 0))}</td>
      </tr>`;
    })
    .join("");
}

function renderRPCRankChart(list) {
  if (!state.charts.rpcRank) {
    return;
  }
  const labels = list.slice(0, 10).map((item) => item.routerId || "-");
  const values = list.slice(0, 10).map((item) => Number(item.perMinute || 0));
  state.charts.rpcRank.setOption({
    tooltip: { trigger: "axis" },
    xAxis: { type: "category", data: labels, axisLabel: { rotate: 25 } },
    yAxis: { type: "value" },
    series: [
      {
        type: "bar",
        name: "每分钟 RPC",
        data: values,
      },
    ],
  });
}

function openRpcDebugModal() {
  rpcDebugModal.classList.remove("hidden");
}

function closeRpcDebugModal() {
  rpcDebugModal.classList.add("hidden");
}

function syncRouteOptions(routers) {
  if (targetRouteIdInput.value) {
    return;
  }
  const first = routers.find((item) => item.routeId);
  if (first) {
    targetRouteIdInput.value = first.routeId;
  }
}

async function loadStubs() {
  const routeId = (targetRouteIdInput.value || "").trim();
  if (!routeId) {
    setRpcMessage("请先填写目标 RouteId");
    return;
  }
  setRpcMessage("加载方法中...");
  try {
    const data = await apiGet(`/api/debug/rpc-stubs?routeId=${encodeURIComponent(routeId)}`);
    const stubs = Array.isArray(data.data) ? data.data : [];
    state.stubs = stubs;
    renderStubOptions(stubs);
    setRpcMessage(`方法加载完成，共 ${stubs.length} 个`);
  } catch (error) {
    setRpcMessage(`加载方法失败: ${error.message || error}`);
  }
}

function renderStubOptions(stubs) {
  if (!stubs.length) {
    stubSelect.innerHTML = `<option value="">该节点暂无可调试方法</option>`;
    packetIdInput.value = "";
    return;
  }
  stubSelect.innerHTML = stubs
    .map((stub) => {
      const packetId = Number(stub.packetId) || 0;
      const text = stub.description || `${stub.className || ""}.${stub.methodName || ""}`;
      return `<option value="${packetId}">[${packetId}] ${escapeHtml(text)}</option>`;
    })
    .join("");
  packetIdInput.value = String(Number(stubs[0].packetId) || "");
}

function onStubChange() {
  const packetId = Number(stubSelect.value) || 0;
  packetIdInput.value = packetId > 0 ? String(packetId) : "";
}

async function sendRpc() {
  const targetRouteId = (targetRouteIdInput.value || "").trim();
  const packetId = Number(packetIdInput.value);
  if (!targetRouteId) {
    setRpcMessage("目标 RouteId 不能为空");
    return;
  }
  if (!packetId || packetId <= 0) {
    setRpcMessage("PacketId 必须大于 0");
    return;
  }

  const raw = paramText.value || "";
  const payload = {
    targetRouteId,
    packetId,
    params: raw.length ? [raw] : [],
  };

  setRpcMessage("发送 RPC 中...");
  rpcResult.textContent = "等待结果...";
  try {
    const sendResp = await apiPost("/api/debug/send-rpc", payload);
    const requestId = sendResp?.data?.requestId || "";
    if (!requestId) {
      setRpcMessage("发送成功，但未拿到 requestId");
      return;
    }
    setRpcMessage(`已发送，requestId=${requestId}，正在查询结果...`);
    await pollRpcResult(requestId, 25, 1000);
  } catch (error) {
    setRpcMessage(`发送失败: ${error.message || error}`);
    rpcResult.textContent = String(error.message || error);
  }
}

async function pollRpcResult(requestId, maxTimes, intervalMs) {
  for (let i = 0; i < maxTimes; i += 1) {
    try {
      const data = await apiGet(`/api/debug/rpc-result?requestId=${encodeURIComponent(requestId)}`);
      if (data.success === true) {
        rpcResult.textContent = JSON.stringify(data.data, null, 2);
        setRpcMessage(`RPC 响应已返回，requestId=${requestId}`);
        return;
      }
    } catch (error) {
      if (!String(error.message || "").includes("结果尚未就绪")) {
        rpcResult.textContent = String(error.message || error);
      }
    }
    await sleep(intervalMs);
  }
  setRpcMessage(`超时未返回结果，requestId=${requestId}`);
}

function updateHistory(values) {
  const now = new Date();
  const label = `${pad2(now.getHours())}:${pad2(now.getMinutes())}:${pad2(now.getSeconds())}`;
  const max = 30;

  pushHistory(state.history.labels, label, max);
  pushHistory(state.history.totalRequests, values.totalRequests, max);
  pushHistory(state.history.totalConnections, values.totalConnections, max);
  pushHistory(state.history.onlineRouters, values.onlineRouters, max);
  pushHistory(state.history.memoryUsagePercent, values.memoryUsagePercent, max);

  state._memoryUsed = values.memoryUsed;
  state._memoryMax = values.memoryMax;
}

function pushHistory(arr, value, max) {
  arr.push(value);
  if (arr.length > max) {
    arr.shift();
  }
}

function initCharts() {
  if (!window.echarts) {
    return;
  }
  state.charts.request = echarts.init(document.getElementById("requestChart"));
  state.charts.memory = echarts.init(document.getElementById("memoryChart"));
  state.charts.online = echarts.init(document.getElementById("onlineChart"));
  state.charts.rpcRank = echarts.init(document.getElementById("rpcRankChart"));
  state.charts.memoryUsageTrend = echarts.init(document.getElementById("memoryUsageTrendChart"));
  state.charts.countryDist = echarts.init(document.getElementById("countryDistChart"));
  state.charts.modeDist = echarts.init(document.getElementById("modeDistChart"));
  state.charts.stubTop = echarts.init(document.getElementById("stubTopChart"));
}

function refreshCharts() {
  if (!state.charts.request || !state.charts.memory || !state.charts.online || !state.charts.memoryUsageTrend) {
    return;
  }

  state.charts.request.setOption({
    tooltip: { trigger: "axis" },
    xAxis: { type: "category", data: state.history.labels },
    yAxis: { type: "value" },
    series: [
      {
        type: "line",
        name: "每分钟请求数",
        smooth: true,
        data: state.history.totalRequests,
      },
    ],
  });

  const used = Number(state._memoryUsed || 0);
  const max = Number(state._memoryMax || 0);
  const free = Math.max(0, max - used);
  state.charts.memory.setOption({
    tooltip: { trigger: "item" },
    series: [
      {
        type: "pie",
        radius: ["45%", "72%"],
        data: [
          { value: used, name: "已用" },
          { value: free, name: "剩余" },
        ],
      },
    ],
  });

  state.charts.online.setOption({
    tooltip: { trigger: "axis" },
    legend: { data: ["历史连接", "在线路由"] },
    xAxis: { type: "category", data: state.history.labels },
    yAxis: { type: "value" },
    series: [
      {
        type: "line",
        name: "历史连接",
        smooth: true,
        data: state.history.totalConnections,
      },
      {
        type: "line",
        name: "在线路由",
        smooth: true,
        data: state.history.onlineRouters,
      },
    ],
  });

  state.charts.memoryUsageTrend.setOption({
    tooltip: { trigger: "axis" },
    xAxis: { type: "category", data: state.history.labels },
    yAxis: { type: "value", min: 0, max: 100 },
    series: [
      {
        type: "line",
        name: "内存使用率",
        areaStyle: {},
        smooth: true,
        data: state.history.memoryUsagePercent,
      },
    ],
  });
}

function resizeCharts() {
  if (state.charts.request) {
    state.charts.request.resize();
  }
  if (state.charts.memory) {
    state.charts.memory.resize();
  }
  if (state.charts.online) {
    state.charts.online.resize();
  }
  if (state.charts.rpcRank) {
    state.charts.rpcRank.resize();
  }
  if (state.charts.memoryUsageTrend) {
    state.charts.memoryUsageTrend.resize();
  }
  if (state.charts.countryDist) {
    state.charts.countryDist.resize();
  }
  if (state.charts.modeDist) {
    state.charts.modeDist.resize();
  }
  if (state.charts.stubTop) {
    state.charts.stubTop.resize();
  }
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function setMessage(message) {
  appMsg.textContent = message;
}

function setRpcMessage(message) {
  rpcMsg.textContent = message;
}

async function updateAdminPassword() {
  const oldPassword = oldPasswordInput.value || "";
  const newPassword = (newPasswordInput.value || "").trim();
  const confirmPassword = (confirmPasswordInput.value || "").trim();

  if (!oldPassword) {
    setSettingsMessage("旧密码不能为空");
    return;
  }
  if (newPassword.length < 4) {
    setSettingsMessage("新密码长度至少 4 位");
    return;
  }
  if (newPassword !== confirmPassword) {
    setSettingsMessage("两次新密码不一致");
    return;
  }

  setSettingsMessage("更新中...");
  try {
    const resp = await apiPost("/api/system/admin-password", {
      oldPassword,
      newPassword,
    });
    setSettingsMessage(resp.message || "密码更新成功");
    oldPasswordInput.value = "";
    newPasswordInput.value = "";
    confirmPasswordInput.value = "";
    localStorage.removeItem(AUTH_TOKEN_KEY);
    setTimeout(() => redirectToLogin("密码已更新，请重新登录"), 600);
  } catch (error) {
    setSettingsMessage(`更新失败: ${error.message || error}`);
  }
}

function setSettingsMessage(message) {
  settingsMsg.textContent = message;
}

function pad2(v) {
  return String(v).padStart(2, "0");
}

function pad3(v) {
  return String(v).padStart(3, "0");
}

function formatDateTime(ms) {
  const value = Number(ms || 0);
  if (!value) {
    return "-";
  }
  const d = new Date(value);
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())} ${pad2(d.getHours())}:${pad2(d.getMinutes())}:${pad2(d.getSeconds())},${pad3(d.getMilliseconds())}`;
}

function formatDuration(ms) {
  const total = Math.max(0, Math.floor(Number(ms || 0) / 1000));
  const days = Math.floor(total / 86400);
  const hours = Math.floor((total % 86400) / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  return `${days}天${hours}小时${minutes}分`;
}

function formatBytesToMB(bytes) {
  const value = Number(bytes || 0) / (1024 * 1024);
  return `${value.toFixed(2)} MB`;
}

function escapeHtml(raw) {
  return raw
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}
