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

function esc(s) {
  return String(s == null ? "" : s).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;");
}
function errText(e) {
  if (!e) return "未知错误";
  if (typeof e === "string") return e;
  if (e.message) return e.message;
  return JSON.stringify(e);
}
function toast(msg, kind = "info") {
  const el = $("toast");
  el.textContent = String(msg);
  el.className = "toast show" + (kind === "error" ? " error" : "");
  clearTimeout(el._t);
  el._t = setTimeout(() => { el.className = "toast"; }, 5000);
}
function banner(msg, kind = "") {
  const el = $("resultBanner");
  el.textContent = msg;
  el.className = "banner " + kind;
  el.classList.remove("hidden");
}

function currentSettings() {
  return {
    api: $("api").value.trim(),
    secret: $("secret").value,
    group: $("group").value.trim(),
    config_path: $("config").value.trim(),
    mihomo_path: $("mihomo").value.trim(),
    url: $("url").value.trim(),
    rounds: parseInt($("quickRounds").value, 10) || 5,
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
  if ([3, 5, 10].includes(s.rounds)) $("quickRounds").value = String(s.rounds);
  $("subPath").textContent = s.config_path ? "当前订阅: " + s.config_path : "当前没有选择订阅文件";
}

async function bootstrap() {
  try {
    const b = await App().Bootstrap();
    state.controllers = b.controllers || [];
    state.subscriptions = b.subscriptions || [];
    applySettings(b.settings);
    renderControllers();
    renderSubscriptions();
    if (b.mihomo) $("mihomo").value = b.mihomo;
    $("detectError").textContent = b.mihomo_error || "";
    if (state.controllers.length) {
      $("connBadge").textContent = "已检测到 " + state.controllers.length + " 个客户端";
      $("connBadge").className = "badge ok";
    } else {
      $("connBadge").textContent = "未检测到客户端";
      $("connBadge").className = "badge warn";
      $("detectHint").textContent = "没检测到 Clash 控制口。FlClash 需要在设置里开启 external-controller；也可以直接用订阅 + mihomo sidecar 测速（不能自动切换）。";
    }
  } catch (e) {
    toast(errText(e), "error");
  }
}

function renderControllers() {
  const wrap = $("controllers");
  if (!state.controllers.length) { wrap.innerHTML = ""; return; }
  const active = $("api").value.trim();
  wrap.innerHTML = state.controllers.map((c) => {
    const cls = active === c.url ? "chip active" : "chip";
    const label = c.auth_required ? c.url + "（需要 secret）" : c.url + (c.version ? " · " + c.version : "");
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
    const wrap = $("groups");
    if (!groups.length) { wrap.innerHTML = '<p class="muted">控制口没有返回组。</p>'; return; }
    wrap.innerHTML = `<table><thead><tr><th>组</th><th>类型</th><th>当前</th><th>成员</th></tr></thead><tbody>${
      groups.map((g) => `<tr><td>${esc(g.name)}</td><td>${esc(g.type)}</td><td>${esc(g.now || "—")}</td><td>${g.members}</td></tr>`).join("")
    }</tbody></table>`;
    if (!$("group").value) {
      const sel = groups.find((g) => String(g.type).toLowerCase() === "selector") || groups[0];
      if (sel) { $("group").value = sel.name; persistSoon(); }
    }
  } catch (e) { toast(errText(e), "error"); }
}

async function oneClick(doSwitch) {
  const btn = doSwitch ? $("oneClick") : $("testOnly");
  const old = btn.textContent;
  btn.disabled = true;
  btn.textContent = doSwitch ? "优化中…" : "测速中…";
  banner("正在测试节点，请稍候…", "");
  try {
    const s = currentSettings();
    const rounds = parseInt($("quickRounds").value, 10) || 5;
    const res = await App().OneClick(s.api, s.secret, s.group, s.config_path, s.mihomo_path, rounds, doSwitch);
    state.results = res.results || [];
    state.best = res.best || null;
    renderResults();
    const kind = res.switched ? "ok" : (res.source === "sidecar" ? "warn" : "ok");
    banner(res.message || ("推荐 " + (res.best ? res.best.name : "")), kind);
    if (res.best) $("switch").disabled = false;
    $("export").disabled = !s.config_path;
    refreshStatus();
  } catch (e) {
    banner(errText(e), "err");
    toast(errText(e), "error");
  } finally {
    btn.disabled = false;
    btn.textContent = old;
  }
}

function renderResults() {
  const wrap = $("results");
  if (!state.results.length) {
    wrap.innerHTML = '<p class="muted">还没有测试结果。点上面的“一键优化”开始。</p>';
    $("resultMeta").textContent = "";
    return;
  }
  const rows = state.results.map((r, i) => {
    const cls = r.success === r.total ? "ok" : (r.success > 0 ? "warn" : "err");
    const best = state.best && r.name === state.best.name ? "best" : "";
    const selected = state.selected === r.name ? "selected" : "";
    return `<tr data-name="${esc(r.name)}" class="${best} ${selected}">
      <td>${i + 1}</td><td>${esc(r.name)}</td><td>${esc(r.type)}</td>
      <td class="${cls}">${r.success}/${r.total}</td>
      <td>${r.median_ms}ms</td><td>${r.p90_ms}ms</td><td>${r.jitter_ms}ms</td><td>${Number(r.score).toFixed(3)}</td>
    </tr>`;
  }).join("");
  wrap.innerHTML = `<table><thead><tr><th>#</th><th>节点</th><th>类型</th><th>OK</th><th>P50</th><th>P90</th><th>抖动</th><th>评分</th></tr></thead><tbody>${rows}</tbody></table>`;
  $("resultMeta").textContent = state.best ? "推荐: " + state.best.name : "";
  wrap.querySelectorAll("tbody tr").forEach((tr) => tr.addEventListener("click", () => {
    state.selected = tr.dataset.name;
    wrap.querySelectorAll("tbody tr").forEach((x) => x.classList.remove("selected"));
    tr.classList.add("selected");
    $("switch").disabled = false;
  }));
}

async function switchSelected() {
  if (!state.selected) return;
  const group = $("group").value.trim();
  if (!group) return toast("请先在高级设置里填写 Selector 组", "error");
  try {
    await App().Switch($("api").value.trim(), $("secret").value, group, state.selected);
    toast("已切换到 " + state.selected);
    refreshStatus();
  } catch (e) { toast(errText(e), "error"); }
}

async function toggleDaemon() {
  const api = $("api").value.trim();
  if (!api) return toast("自动优化需要先连接客户端（Clash 控制口）", "error");
  try {
    if (state.daemon) {
      await App().StopDaemon();
      state.daemon = false;
      $("daemonToggle").textContent = "开启自动优化";
      $("daemonState").textContent = "未开启";
      toast("已停止后台自动优化");
    } else {
      const interval = parseInt($("daemonInterval").value, 10) || 300;
      const rounds = parseInt($("quickRounds").value, 10) || 5;
      await App().StartDaemon(api, $("secret").value, $("group").value.trim(), interval, rounds, 0.2, 2, 0.8);
      state.daemon = true;
      $("daemonToggle").textContent = "停止自动优化";
      $("daemonState").textContent = "运行中";
      toast("后台自动优化已开启");
    }
  } catch (e) { toast(errText(e), "error"); }
}

function addEvent(e) {
  state.events.unshift(e);
  state.events = state.events.slice(0, 8);
  $("history").innerHTML = state.events.map((x) => `<div>${esc((x.time || "").slice(11, 19))} · ${esc(x.kind)} · ${esc(x.from || "")} → ${esc(x.to || "")} · ${esc(x.reason || "")}</div>`).join("");
}

async function chooseConfig() {
  try {
    const p = await App().ChooseConfig();
    if (p) { $("config").value = p; persistSoon(); $("subPath").textContent = "当前订阅: " + p; }
  } catch (e) { toast(errText(e), "error"); }
}

async function importSub() {
  const url = $("subURL").value.trim();
  if (!url) return toast("请粘贴订阅 URL", "error");
  try {
    toast("正在下载订阅…");
    const path = await App().ImportSubscription(url);
    $("config").value = path;
    $("subPath").textContent = "当前订阅: " + path;
    $("subURL").value = "";
    toast("订阅已导入");
    await bootstrap();
  } catch (e) { toast(errText(e), "error"); }
}

function wire() {
  $("redetect").addEventListener("click", bootstrap);
  $("useSub").addEventListener("click", () => {
    $("config").value = $("subSelect").value;
    $("subPath").textContent = $("config").value ? "当前订阅: " + $("config").value : "当前没有选择订阅文件";
    persistSoon();
  });
  $("importSub").addEventListener("click", importSub);
  $("oneClick").addEventListener("click", () => oneClick(true));
  $("testOnly").addEventListener("click", () => oneClick(false));
  $("daemonToggle").addEventListener("click", toggleDaemon);
  $("refresh").addEventListener("click", refreshStatus);
  $("testAdvanced").addEventListener("click", async () => {
    try {
      const s = currentSettings();
      const rounds = parseInt($("quickRounds").value, 10) || 5;
      state.results = await App().Test(s.api, s.secret, s.config_path, s.mihomo_path, s.url, rounds, s.concurrency, s.timeout_ms);
      state.best = null;
      renderResults();
      toast("测速完成");
    } catch (e) { toast(errText(e), "error"); }
  });
  $("switch").addEventListener("click", switchSelected);
  $("export").addEventListener("click", async () => {
    try {
      const msg = await App().Export("stable.yaml", 0.8);
      toast(msg);
    } catch (e) { toast(errText(e), "error"); }
  });
  document.querySelectorAll("input, select").forEach((el) => el.addEventListener("input", persistSoon));
}

async function init() {
  if (!window.go || !window.go.main || !window.go.main.App) {
    setTimeout(init, 80);
    return;
  }
  wire();
  try { $("version").textContent = await App().Version(); } catch (_) {}
  try { state.daemon = await App().DaemonStatus(); } catch (_) {}
  if (state.daemon) {
    $("daemonToggle").textContent = "停止自动优化";
    $("daemonState").textContent = "运行中";
  }
  if (window.runtime && window.runtime.EventsOn) {
    window.runtime.EventsOn("nodepulse:daemon", addEvent);
  }
  await bootstrap();
  if ($("api").value) refreshStatus();
}

document.addEventListener("DOMContentLoaded", init);
