const $ = (id) => document.getElementById(id);
const KEY = "nodepulse.web.settings.v1";
let results = [];
let selected = null;

function toast(msg, kind = "info") {
  const el = $("toast");
  el.textContent = String(msg);
  el.className = "toast show" + (kind === "error" ? " error" : "");
  clearTimeout(el._t);
  el._t = setTimeout(() => { el.className = "toast"; }, 4500);
}

async function call(path, opts) {
  const resp = await fetch(path, opts);
  const data = await resp.json().catch(() => null);
  if (!resp.ok) throw new Error((data && data.error) || resp.statusText);
  return data;
}

function esc(s) {
  return String(s == null ? "" : s).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;");
}

function settings() {
  return {
    api: $("api").value.trim(), secret: $("secret").value, group: $("group").value.trim(),
    config: $("config").value.trim(), mihomo: $("mihomo").value.trim(),
    url: $("url").value.trim(), rounds: $("rounds").value, conc: $("conc").value,
  };
}
function save() { localStorage.setItem(KEY, JSON.stringify(settings())); }
function restore() {
  let s = {};
  try { s = JSON.parse(localStorage.getItem(KEY) || "{}"); } catch (_) {}
  for (const [k, id] of Object.entries({ api: "api", secret: "secret", group: "group", config: "config", mihomo: "mihomo", url: "url", rounds: "rounds", conc: "conc" })) {
    if (s[k]) $(id).value = s[k];
  }
}

async function refreshStatus() {
  try {
    const api = encodeURIComponent($("api").value.trim());
    const secret = encodeURIComponent($("secret").value);
    if (!api) return toast("请填写 Clash 控制口", "error");
    const groups = await call(`/api/status?api=${api}&secret=${secret}`);
    const wrap = $("groups");
    if (!groups.length) {
      wrap.innerHTML = '<p class="muted">控制口没有返回组。</p>';
    } else {
      wrap.innerHTML = `<table><thead><tr><th>组</th><th>类型</th><th>当前</th><th>成员</th></tr></thead><tbody>${
        groups.map((g) => `<tr><td>${esc(g.name)}</td><td>${esc(g.type)}</td><td>${esc(g.now || "—")}</td><td>${g.members}</td></tr>`).join("")
      }</tbody></table>`;
      if (!$("group").value) {
        const sel = groups.find((g) => String(g.type).toLowerCase() === "selector") || groups[0];
        if (sel) $("group").value = sel.name;
      }
    }
    save();
  } catch (e) { toast(e.message, "error"); }
}

function renderResults(list) {
  results = list || [];
  const wrap = $("results");
  if (!results.length) { wrap.innerHTML = '<p class="muted">还没有测试结果。</p>'; return; }
  const rows = results.map((r, i) => {
    const cls = r.success === r.total ? "ok" : (r.success ? "warn" : "err");
    return `<tr data-name="${esc(r.name)}" class="${selected === r.name ? "selected" : ""}">
      <td>${i + 1}</td><td>${esc(r.name)}</td><td>${esc(r.type)}</td>
      <td class="${cls}">${r.success}/${r.total}</td>
      <td>${r.median_ms}ms</td><td>${r.p90_ms}ms</td><td>${r.jitter_ms}ms</td><td>${Number(r.score).toFixed(3)}</td>
    </tr>`;
  }).join("");
  wrap.innerHTML = `<table><thead><tr><th>#</th><th>节点</th><th>类型</th><th>OK</th><th>P50</th><th>P90</th><th>抖动</th><th>评分</th></tr></thead><tbody>${rows}</tbody></table>`;
  wrap.querySelectorAll("tbody tr").forEach((tr) => tr.addEventListener("click", () => {
    selected = tr.dataset.name;
    wrap.querySelectorAll("tbody tr").forEach((x) => x.classList.remove("selected"));
    tr.classList.add("selected");
    $("switch").disabled = false;
  }));
}

async function runTest() {
  const btn = $("start");
  btn.disabled = true;
  btn.textContent = "测试中…";
  $("testMsg").textContent = "测试中，请稍候…";
  try {
    const body = {
      api: $("api").value.trim(), secret: $("secret").value,
      config_path: $("config").value.trim(), mihomo_path: $("mihomo").value.trim(),
      url: $("url").value.trim(), rounds: parseInt($("rounds").value, 10) || 5,
      concurrency: parseInt($("conc").value, 10) || 8, timeout_ms: 5000,
    };
    const list = await call("/api/test", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
    selected = null;
    $("switch").disabled = true;
    renderResults(list);
    $("testMsg").textContent = `完成 ${list.length} 个节点`;
    toast(`测试完成：${list.length} 个节点`);
    save();
  } catch (e) { $("testMsg").textContent = ""; toast(e.message, "error"); }
  finally { btn.disabled = false; btn.textContent = "开始测试"; }
}

async function switchNode() {
  if (!selected) return;
  const group = $("group").value.trim();
  if (!group) return toast("请填写 Selector 组", "error");
  try {
    await call("/api/switch", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ api: $("api").value.trim(), secret: $("secret").value, group, node: selected }),
    });
    toast(`已切换到 ${selected}`);
    refreshStatus();
  } catch (e) { toast(e.message, "error"); }
}

async function loadHistory() {
  try {
    const events = await call("/api/history");
    const wrap = $("history");
    if (!events || !events.length) { wrap.innerHTML = '<p class="muted">暂无历史。</p>'; return; }
    const rows = events.slice().reverse().map((e) => `<tr><td>${esc(e.time)}</td><td>${esc(e.kind)}</td><td>${esc(e.from)}</td><td>${esc(e.to)}</td><td>${esc(e.reason)}</td></tr>`).join("");
    wrap.innerHTML = `<table><thead><tr><th>时间</th><th>类型</th><th>从</th><th>到</th><th>原因</th></tr></thead><tbody>${rows}</tbody></table>`;
  } catch (e) { toast(e.message, "error"); }
}

async function loadLatest() {
  try {
    const rf = await call("/api/results");
    if (rf && rf.results) renderResults(rf.results);
  } catch (_) { /* no results yet */ }
}

document.addEventListener("DOMContentLoaded", () => {
  restore();
  $("refresh").addEventListener("click", refreshStatus);
  $("start").addEventListener("click", runTest);
  $("switch").addEventListener("click", switchNode);
  $("reloadHistory").addEventListener("click", loadHistory);
  document.querySelectorAll("input").forEach((el) => el.addEventListener("input", save));
  loadLatest();
  loadHistory();
});
