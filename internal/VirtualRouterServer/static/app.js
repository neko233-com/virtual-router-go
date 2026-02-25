const state = {
  token: "",
  activeTab: "home",
  stubs: [],
  history: {
    labels: [],
    totalRequests: [],
    totalConnections: [],
    onlineRouters: [],
  },
  charts: {
    request: null,
    memory: null,
    online: null,
  },
};

const passwordInput = document.getElementById("password");
const loginBtn = document.getElementById("loginBtn");
const refreshBtn = document.getElementById("refreshBtn");
const loginMsg = document.getElementById("loginMsg");
const tabs = document.getElementById("tabs");

const stats = document.getElementById("stats");
const routersBody = document.getElementById("routersBody");
const settingsInfo = document.getElementById("settingsInfo");
const logLimitInput = document.getElementById("logLimit");
const loadLogsBtn = document.getElementById("loadLogsBtn");
const logOutput = document.getElementById("logOutput");

const targetRouteIdInput = document.getElementById("targetRouteId");
const packetIdInput = document.getElementById("packetId");
const loadStubsBtn = document.getElementById("loadStubsBtn");
const stubSelect = document.getElementById("stubSelect");
const paramText = document.getElementById("paramText");
const sendRpcBtn = document.getElementById("sendRpcBtn");
const rpcMsg = document.getElementById("rpcMsg");
const rpcResult = document.getElementById("rpcResult");

loginBtn.addEventListener("click", login);
refreshBtn.addEventListener("click", loadAll);
tabs.addEventListener("click", onTabClick);
loadLogsBtn.addEventListener("click", loadLogs);

loadStubsBtn.addEventListener("click", loadStubs);
sendRpcBtn.addEventListener("click", sendRpc);
stubSelect.addEventListener("change", onStubChange);

window.addEventListener("resize", resizeCharts);

initCharts();

async function login() {
  setMessage("登录中...");
  try {
    const resp = await fetch("/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ password: passwordInput.value || "" }),
    });
    const data = await resp.json();
    if (!resp.ok || !data.success) {
      setMessage(data.message || "登录失败");
      return;
    }

    state.token = data.data.token;
    refreshBtn.disabled = false;
    loadStubsBtn.disabled = false;
    sendRpcBtn.disabled = false;
    stubSelect.disabled = false;
    loadLogsBtn.disabled = false;

    setMessage("登录成功");
    await loadAll();
    startAutoRefresh();
  } catch (error) {
    setMessage(`登录异常: ${error.message || error}`);
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
    if (state.activeTab === "logs") {
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
    const [statusRes, metricsRes, connRes, routersRes, monitorRes, viewersRes, rpcStatsRes] = await Promise.all([
      apiGet("/api/status"),
      apiGet("/api/metrics"),
      apiGet("/api/connections"),
      apiGet("/api/routers"),
      apiGet("/api/monitor-stats"),
      apiGet("/api/viewers"),
      apiGet("/api/rpc-stats"),
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
    const routersData = routersRes.data || {};
    const routers = routersData.routers || routersData.list || [];

    renderStats([
      ["运行时长(ms)", serverInfo.uptime || 0],
      ["总请求数", connData.totalRequests || 0],
      ["历史连接数", connData.totalConnections || 0],
      ["在线路由", routersData.online || routers.length || 0],
      ["活跃查看者", viewers.activeViewers || 0],
      ["累计字节", rpcTotal.bytes || 0],
    ]);
    renderRouters(routers);
    syncRouteOptions(routers);
    renderSettings({
      osName: systemInfo.osName || "-",
      cpuCores: systemInfo.processors || 0,
      routerPort: routerInfo.port || 0,
      monitorPort: routerInfo.monitorPort || 0,
      cpuUsage: cpu.usage || 0,
      memoryUsed: memory.used || 0,
      memoryMax: memory.max || 0,
      requestsLastMinute: monitor.requestsLastMinute || 0,
    });

    updateHistory({
      totalRequests: connData.totalRequests || 0,
      totalConnections: connData.totalConnections || 0,
      onlineRouters: routersData.online || routers.length || 0,
      memoryUsed: memory.used || 0,
      memoryMax: memory.max || 0,
    });
    refreshCharts();

    if (!isSilent) {
      setMessage(`刷新成功，节点数: ${routers.length}`);
    }
  } catch (error) {
    setMessage(`加载失败: ${error.message || error}`);
  }
}

async function loadLogs() {
  if (!state.token) {
    return;
  }
  const limit = Math.min(1000, Math.max(10, Number(logLimitInput.value || 200)));
  try {
    const data = await apiGet(`/api/logs?limit=${limit}`);
    const lines = data?.data?.lines || [];
    logOutput.textContent = lines.length ? lines.join("\n") : "暂无日志";
  } catch (error) {
    logOutput.textContent = `日志加载失败: ${error.message || error}`;
  }
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
  ];
  settingsInfo.innerHTML = pairs
    .map(([k, v]) => `<div class="kv-k">${escapeHtml(String(k))}</div><div class="kv-v">${escapeHtml(String(v))}</div>`)
    .join("");
}

function renderRouters(list) {
  if (!list.length) {
    routersBody.innerHTML = `<tr><td colspan="4">暂无在线路由节点</td></tr>`;
    return;
  }
  routersBody.innerHTML = list
    .map((r) => {
      const routeId = r.routeId || "-";
      const mode = r.rpcMode || "-";
      const addr = r.address || "-";
      const status = r.status || (r.connected ? "ONLINE" : "OFFLINE");
      return `<tr><td>${escapeHtml(routeId)}</td><td>${escapeHtml(mode)}</td><td>${escapeHtml(addr)}</td><td>${escapeHtml(status)}</td></tr>`;
    })
    .join("");
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
}

function refreshCharts() {
  if (!state.charts.request || !state.charts.memory || !state.charts.online) {
    return;
  }

  state.charts.request.setOption({
    tooltip: { trigger: "axis" },
    xAxis: { type: "category", data: state.history.labels },
    yAxis: { type: "value" },
    series: [
      {
        type: "line",
        name: "总请求数",
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
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function setMessage(message) {
  loginMsg.textContent = message;
}

function setRpcMessage(message) {
  rpcMsg.textContent = message;
}

function pad2(v) {
  return String(v).padStart(2, "0");
}

function escapeHtml(raw) {
  return raw
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}
