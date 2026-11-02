const $ = (id) => document.getElementById(id);
const App = () => window.go && window.go.main && window.go.main.App;

const state = {
  controllers: [],
  subscriptions: [],
  results: [],
  selected: null,
  best: null,
  daemon: false,
  events: [],
};

const viewMeta = {
  dashboard: ["仪表盘", "找到最稳的节点，自动保持连接质量"],
  nodes: ["节点", "按成功率、P90 和抖动综合排序"],
  settings: ["设置", "连接、订阅与高级参数"],
  history: ["历史", "自动优化与切换记录"],
};

function esc(s) {
  return String(s == null ? "" : s).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;");
}
function errText(e) {
  if (!e) return "未知错误";
  if (typeof e === "string") return e;
  if (e.message) return e.message;
  return JSON.stringify(e);
}
function snack(msg, kind = "info") {
  const el = $("snackbar");
  el.textContent = String(msg);
  el.className = "snackbar show" + (kind === "error" ? " error" : "");
  clearTimeout(el._t);
  el._t = setTimeout(() => { el.className = "snackbar"; }, 5000);
}

function currentSettings() {
  return {
    api: $("api").value.trim(),
    secret: $("secret").value,
    group: $("group").value.trim(),
    config_path: $("config").value.trim(),
    mihomo_path: $("mihomo").value.trim(),
    url: $("url").value.trim(),
    rounds: parseInt($("quickRounds")?.value || "3", 10) || 3,
    concurrency: parseInt($("conc").value, 10) || 8,
    timeout_ms: parseInt($("timeout").value, 10) || 5000,
  };
}

let saveTimer = null;
function persistSoon() {
  clearTimeout(saveTimer);
  saveTimer = setTimeout(async () => {
    try { await App().SaveSettings(currentSettings()); } catch (_) {}
  }, 350);
}

function applySettings(s) {
  if (!s) return;
  $("api").value = s.api || "";
  $("secret").value = s.secret || "";
  $("group").value = s.group || "";
  $("config").value = s.config_path || "";
  $("mihomo").value = s.mihomo_path || "";
  $("url").value = s.url || "https://www.gstatic.com/generate_204";
  $("conc").value = s.concurrency || 8;
  $("timeout").value = s.timeout_ms || 5000;
  if ([3, 5, 10].includes(Number(s.rounds))) $("quickRounds").value = String(s.rounds);
  $("subPath").textContent = s.config_path ? "当前订阅: " + s.config_path : "当前没有选择订阅文件";
}

async function bootstrap() {
  try {
    const b = await App().Bootstrap();
    state.controllers = b.controllers || [];
    state.subscriptions = b.subscriptions || [];
    applySettings(b.settings);
    if (b.mihomo) $("mihomo").value = b.mihomo;
    $("detectError") && ($("detectError").textContent = b.mihomo_error || "");
    renderControllers();
    renderSubscriptions();
    if (state.controllers.length) {
      $("connBadge").textContent = "已连接 " + state.controllers.length + " 个客户端";
      $("connBadge").className = "status-chip ok";
    } else {
      $("connBadge").textContent = "未检测到客户端";
      $("connBadge").className = "status-chip warn";
    }
    if ($("api").value) refreshStatus();
  } catch (e) {
    snack(errText(e), "error");
  }
}

function renderControllers() {
  const wrap = $("controllers");
  if (!wrap) return;
  if (!state.controllers.length) {
    wrap.innerHTML = '<span class="muted">没检测到控制口。开启方式见下方说明，或只用订阅测速。</span>';
    return;
  }
  const active = $("api").value.trim();
  const hasSecret = !!$("secret").value;
  wrap.innerHTML = state.controllers.map((c) => {
    const cls = active === c.url ? "chip active" : "chip";
    const needSecret = c.auth_required && !hasSecret;
    const label = needSecret ? c.url + "（需要 secret）" : c.url + (c.version ? " · " + c.version : "");
    return `<button class="${cls}" data-url="${esc(c.url)}"><span class="dot"></span>${esc(label)}</button>`;
  }).join("");
  wrap.querySelectorAll("button").forEach((btn) => btn.addEventListener("click", () => {
    $("api").value = btn.dataset.url;
    renderControllers();
    persistSoon();
    refreshStatus();
  }));
}

function renderSubscriptions() {
  const sel = $("subSelect");
  if (!state.subscriptions.length) {
    sel.innerHTML = '<option value="">没有自动找到订阅文件</option>';
    return;
  }
  const cur = $("config").value.trim();
  sel.innerHTML = state.subscriptions.map((p) => `<option value="${esc(p)}" ${p === cur ? "selected" : ""}>${esc(p)}</option>`).join("");
}

async function refreshStatus() {
  const api = $("api").value.trim();
  if (!api) return;
  try {
    const groups = await App().Status(api, $("secret").value);
    const groupName = $("group").value.trim() || (groups.find((g) => String(g.type).toLowerCase() === "selector") || groups[0] || {}).name;
    if (!$("group").value && groupName) { $("group").value = groupName; persistSoon(); }
    const g = groups.find((x) => x.name === groupName);
    if (g) {
      $("currentNode").textContent = g.now || "—";
      $("currentMeta").textContent = `${g.name} · ${g.type} · ${g.members} 个成员`;
    }
    renderGroups(groups);
  } catch (e) {
    snack(errText(e), "error");
  }
}

function renderGroups(groups) {
  const wrap = $("groups");
  if (!wrap) return;
  if (!groups.length) { wrap.innerHTML = '<p class="muted">控制口没有返回组。</p>'; return; }
  wrap.innerHTML = `<div class="node-head node-row" style="grid-template-columns:34px minmax(160px,1.5fr) 100px minmax(120px,1fr)"><div>#</div><div>组</div><div>类型</div><div class="node-num">当前</div></div>` +
    groups.map((g, i) => `<div class="node-row" style="grid-template-columns:34px minmax(160px,1.5fr) 100px minmax(120px,1fr)">
      <div class="node-rank">${i + 1}</div>
      <div class="node-name">${esc(g.name)}</div>
      <div class="node-type">${esc(g.type)}</div>
      <div class="node-num">${esc(g.now || "—")} · ${g.members}</div>
    </div>`).join("");
}

async function oneClick(doSwitch) {
  const btn = doSwitch ? $("oneClick") : $("testOnly");
  const old = btn.innerHTML;
  btn.disabled = true;
  btn.textContent = doSwitch ? "优化中…" : "测速中…";
  snack(doSwitch ? "正在测速并选择最优节点…" : "正在测速…");
  try {
    const s = currentSettings();
    const rounds = parseInt($("quickRounds").value, 10) || 3;
    const res = await App().OneClick(s.api, s.secret, s.group, s.config_path, s.mihomo_path, rounds, doSwitch);
    state.results = res.results || [];
    state.best = res.best || null;
    renderResults();
    renderBest(res);
    const kind = res.source === "sidecar" ? "" : "ok";
    snack(res.message || ("推荐 " + (res.best ? res.best.name : "")), kind);
    $("switch").disabled = !state.selected;
    $("export").disabled = !s.config_path;
    refreshStatus();
  } catch (e) {
    snack(errText(e), "error");
  } finally {
    btn.disabled = false;
    btn.innerHTML = old;
  }
}

function renderBest(res) {
  const score = res.best ? Number(res.best.score) : 0;
  const p = Math.max(0, Math.min(100, Math.round(score * 100)));
  $("scoreRing").style.setProperty("--p", p);
  $("scoreValue").textContent = res.best ? p : "--";
  $("bestName").textContent = res.best ? res.best.name : "";
  if (res.current) {
    $("currentNode").textContent = res.current;
    $("currentMeta").textContent = `${res.group || "自动"} · ${res.source === "controller" ? "客户端" : "sidecar"}`;
  }
}

function resultRows() {
  if (!state.results.length) return '<p class="muted">还没有测试结果。点“一键优化”开始。</p>';
  const head = `<div class="node-head node-row">
    <div>#</div><div>节点</div><div>类型</div>
    <div class="node-num">OK</div><div class="node-num">P50</div><div class="node-num">P90</div><div class="node-num">抖动</div><div class="node-score">评分</div>
  </div>`;
  const rows = state.results.map((r, i) => {
    const cls = r.success === r.total ? "ok" : (r.success > 0 ? "warn" : "err");
    const best = state.best && r.name === state.best.name ? "best" : "";
    const selected = state.selected === r.name ? "selected" : "";
    const dead = r.success === 0;
    const p50 = dead ? "—" : r.median_ms + "ms";
    const p90 = dead ? "—" : r.p90_ms + "ms";
    const jitter = dead ? "—" : r.jitter_ms + "ms";
    return `<div class="node-row ${best} ${selected}" data-name="${esc(r.name)}">
      <div class="node-rank">${i + 1}</div>
      <div class="node-name" title="${esc(r.name)}">${esc(r.name)}</div>
      <div class="node-type">${esc(r.type)}</div>
      <div class="node-num ${cls}">${r.success}/${r.total}</div>
      <div class="node-num">${p50}</div>
      <div class="node-num">${p90}</div>
      <div class="node-num">${jitter}</div>
      <div class="node-score">${Number(r.score).toFixed(3)}</div>
    </div>`;
  }).join("");
  return head + rows;
}

function bindRows(wrap) {
  wrap.querySelectorAll(".node-row[data-name]").forEach((row) => row.addEventListener("click", () => {
    state.selected = row.dataset.name;
    wrap.querySelectorAll(".node-row").forEach((x) => x.classList.remove("selected"));
    row.classList.add("selected");
    $("switch").disabled = false;
  }));
}

function renderResults() {
  const wrap = $("results");
  wrap.innerHTML = resultRows();
  bindRows(wrap);
  $("resultMeta").textContent = state.best ? "推荐: " + state.best.name : "";
  renderNodesList();
}

function renderNodesList() {
  const wrap = $("nodesList");
  if (!wrap) return;
  const q = ($("search")?.value || "").trim().toLowerCase();
  const rows = q ? state.results.filter((r) => r.name.toLowerCase().includes(q)) : state.results;
  if (!rows.length) {
    wrap.innerHTML = '<p class="muted">没有匹配的节点。</p>';
    return;
  }
  const saved = state.results;
  state.results = rows;
  wrap.innerHTML = resultRows();
  bindRows(wrap);
  state.results = saved;
}

async function switchSelected() {
  if (!state.selected) return;
  const group = $("group").value.trim();
  if (!group) return snack("请先在设置里填写 Selector 组", "error");
  try {
    await App().Switch($("api").value.trim(), $("secret").value, group, state.selected);
    snack("已切换到 " + state.selected, "ok");
    refreshStatus();
  } catch (e) { snack(errText(e), "error"); }
}

async function toggleDaemon(enable) {
  try {
    if (enable) {
      const api = $("api").value.trim();
      if (!api) { $("daemonToggle").checked = false; return snack("自动优化需要先连接客户端", "error"); }
      const interval = parseInt($("daemonInterval").value, 10) || 300;
      const rounds = parseInt($("quickRounds").value, 10) || 3;
      await App().StartDaemon(api, $("secret").value, $("group").value.trim(), interval, rounds, 0.2, 2, 0.8);
      state.daemon = true;
      $("daemonState").textContent = "运行中";
      snack("后台自动优化已开启", "ok");
    } else {
      await App().StopDaemon();
      state.daemon = false;
      $("daemonState").textContent = "未开启";
      snack("已停止后台自动优化");
    }
  } catch (e) {
    $("daemonToggle").checked = state.daemon;
    snack(errText(e), "error");
  }
}

function addEvent(e) {
  state.events.unshift(e);
  state.events = state.events.slice(0, 30);
  renderHistory();
}
function renderHistory() {
  const wrap = $("history");
  if (!state.events.length) { wrap.innerHTML = '<p class="muted">暂无事件。开启自动优化后这里会显示切换记录。</p>'; return; }
  wrap.innerHTML = state.events.map((x) => `<div class="ev">
    <div>${esc((x.time || "").slice(11, 19))}</div>
    <div>${esc(x.kind || "")}</div>
    <div>${esc(x.from || "")} → ${esc(x.to || "")} · ${esc(x.reason || "")}</div>
  </div>`).join("");
}

async function importSub() {
  const url = $("subURL").value.trim();
  if (!url) return snack("请粘贴订阅 URL", "error");
  try {
    snack("正在下载订阅…");
    const path = await App().ImportSubscription(url);
    $("config").value = path;
    $("subPath").textContent = "当前订阅: " + path;
    $("subURL").value = "";
    snack("订阅已导入", "ok");
    await bootstrap();
  } catch (e) { snack(errText(e), "error"); }
}

function switchView(name) {
  document.querySelectorAll(".view").forEach((v) => v.classList.toggle("active", v.id === "view-" + name));
  document.querySelectorAll(".rail-item[data-view]").forEach((b) => b.classList.toggle("active", b.dataset.view === name));
  const meta = viewMeta[name] || ["", ""];
  $("viewTitle").textContent = meta[0];
  $("viewSub").textContent = meta[1];
  if (name === "nodes") renderNodesList();
}

function initTheme() {
  const saved = localStorage.getItem("nodepulse.theme");
  if (saved === "light" || saved === "dark") document.documentElement.dataset.theme = saved;
  updateThemeIcon();
}
function updateThemeIcon() {
  const dark = document.documentElement.dataset.theme !== "light";
  $("themeToggle").querySelector("use").setAttribute("href", dark ? "#i-sun" : "#i-moon");
}
function toggleTheme() {
  const next = document.documentElement.dataset.theme === "light" ? "dark" : "light";
  document.documentElement.dataset.theme = next;
  localStorage.setItem("nodepulse.theme", next);
  updateThemeIcon();
}

function wire() {
  document.querySelectorAll(".rail-item[data-view]").forEach((b) => b.addEventListener("click", () => switchView(b.dataset.view)));
  $("themeToggle").addEventListener("click", toggleTheme);
  $("oneClick").addEventListener("click", () => oneClick(true));
  $("testOnly").addEventListener("click", () => oneClick(false));
  $("daemonToggle").addEventListener("change", (e) => toggleDaemon(e.target.checked));
  $("importSub").addEventListener("click", importSub);
  $("useSub").addEventListener("click", () => {
    $("config").value = $("subSelect").value;
    $("subPath").textContent = $("config").value ? "当前订阅: " + $("config").value : "当前没有选择订阅文件";
    persistSoon();
  });
  $("refresh").addEventListener("click", refreshStatus);
  $("saveSettings").addEventListener("click", async () => {
    try { await App().SaveSettings(currentSettings()); snack("设置已保存", "ok"); } catch (e) { snack(errText(e), "error"); }
  });
  $("testAdvanced").addEventListener("click", async () => {
    try {
      const s = currentSettings();
      state.results = await App().Test(s.api, s.secret, s.config_path, s.mihomo_path, s.url, s.rounds, s.concurrency, s.timeout_ms);
      state.best = null;
      renderResults();
      snack("测速完成", "ok");
      switchView("nodes");
    } catch (e) { snack(errText(e), "error"); }
  });
  $("switch").addEventListener("click", switchSelected);
  $("export").addEventListener("click", async () => {
    try { snack(await App().Export("stable.yaml", 0.8), "ok"); } catch (e) { snack(errText(e), "error"); }
  });
  $("search").addEventListener("input", renderNodesList);
  document.querySelectorAll("input, select").forEach((el) => el.addEventListener("input", persistSoon));
}

async function init() {
  if (!window.go || !window.go.main || !window.go.main.App) {
    setTimeout(init, 80);
    return;
  }
  initTheme();
  wire();
  try { $("version").textContent = "v" + await App().Version(); } catch (_) {}
  try { state.daemon = await App().DaemonStatus(); } catch (_) {}
  $("daemonToggle").checked = state.daemon;
  $("daemonState").textContent = state.daemon ? "运行中" : "未开启";
  if (window.runtime && window.runtime.EventsOn) window.runtime.EventsOn("nodepulse:daemon", addEvent);
  renderHistory();
  await bootstrap();
}

document.addEventListener("DOMContentLoaded", init);
