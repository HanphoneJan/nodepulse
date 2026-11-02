import { html } from "./html.js";
import { render } from "../vendor/preact.module.js";
import { useCallback, useEffect, useRef, useState } from "../vendor/hooks.module.js";
import { api } from "./api.js";
import { subscribeDaemon } from "./events.js";
import {
  currentMode,
  currentSeed,
  applyMode,
  applySeed,
  bootTheme,
  modeLabel,
  nextMode,
  normalizeSeed,
  setMode,
  DEFAULT_SEED,
} from "./theme.js";
import { Icon, IconButton, Snackbar } from "./components.js";
import { Dashboard, Nodes, Settings, History } from "./views.js";
import { timeAgo } from "./format.js";

const LAST_TEST_KEY = "nodepulse.lastTest.v1";
const NAV = [
  { view: "dashboard", label: "仪表盘", icon: "dashboard" },
  { view: "nodes", label: "节点", icon: "list" },
  { view: "settings", label: "设置", icon: "settings" },
  { view: "history", label: "历史", icon: "history" },
];
const VIEW_META = {
  dashboard: ["仪表盘", "找到最稳的节点，自动保持连接质量"],
  nodes: ["节点", "按成功率、P90 和抖动综合排序"],
  settings: ["设置", "连接、订阅与高级参数"],
  history: ["历史", "自动优化与切换记录"],
};

function defaultSettings() {
  return {
    api: "",
    secret: "",
    group: "",
    configPath: "",
    mihomoPath: "",
    url: "https://www.gstatic.com/generate_204",
    rounds: 5,
    concurrency: 8,
    timeoutMs: 5000,
  };
}

function mapSettings(raw) {
  return {
    api: raw.api || "",
    secret: raw.secret || "",
    group: raw.group || "",
    configPath: raw.config_path || "",
    mihomoPath: raw.mihomo_path || "",
    url: raw.url || "https://www.gstatic.com/generate_204",
    rounds: Number(raw.rounds) || 5,
    concurrency: Number(raw.concurrency) || 8,
    timeoutMs: Number(raw.timeout_ms) || 5000,
  };
}

function errText(error) {
  if (!error) return "未知错误";
  if (typeof error === "string") return error;
  if (error.message) return error.message;
  try { return JSON.stringify(error); } catch (_) { return "未知错误"; }
}

function sortEvents(list) {
  return [...(list || [])]
    .sort((a, b) => String(b.time || "").localeCompare(String(a.time || "")))
    .slice(0, 50);
}

function readLastTest() {
  try { return localStorage.getItem(LAST_TEST_KEY) || ""; } catch (_) { return ""; }
}

function writeLastTest(value) {
  try { localStorage.setItem(LAST_TEST_KEY, value); } catch (_) {}
}

function App() {
  const [view, setView] = useState("dashboard");
  const [settings, setSettings] = useState(defaultSettings);
  const [controllers, setControllers] = useState([]);
  const [subscriptions, setSubscriptions] = useState([]);
  const [status, setStatus] = useState({ connected: false, current: "", groupName: "", members: 0, error: "" });
  const [groups, setGroups] = useState([]);
  const [results, setResults] = useState([]);
  const [best, setBest] = useState(null);
  const [selected, setSelected] = useState(null);
  const [daemon, setDaemon] = useState(false);
  const [daemonInterval, setDaemonInterval] = useState(300);
  const [events, setEvents] = useState([]);
  const [busy, setBusy] = useState(false);
  const [busyKind, setBusyKind] = useState("");
  const [elapsed, setElapsed] = useState(0);
  const [lastTestAt, setLastTestAt] = useState(readLastTest);
  const [snack, setSnack] = useState(null);
  const [subURL, setSubURL] = useState("");
  const [version, setVersion] = useState("");
  const [themeMode, setThemeMode] = useState(currentMode);
  const [seed, setSeed] = useState(currentSeed);
  const [scrolled, setScrolled] = useState(false);
  const [, setClockTick] = useState(0);

  const settingsRef = useRef(settings);
  const seedRef = useRef(seed);
  const hydratedRef = useRef(false);
  const saveTimer = useRef(null);
  const seedTimer = useRef(null);
  const snackTimer = useRef(null);
  const contentRef = useRef(null);

  settingsRef.current = settings;
  seedRef.current = seed;

  const notify = useCallback((text, kind = "info") => {
    clearTimeout(snackTimer.current);
    setSnack({ text: String(text), kind, id: Date.now() });
    snackTimer.current = setTimeout(() => setSnack(null), kind === "error" ? 8000 : 5000);
  }, []);

  function updateSettings(patch) {
    setSettings((previous) => {
      const next = { ...previous, ...patch };
      settingsRef.current = next;
      return next;
    });
  }

  const refreshStatus = useCallback(async (override) => {
    const current = override || settingsRef.current;
    if (!current.api) {
      setGroups([]);
      setStatus({ connected: false, current: "", groupName: current.group || "", members: 0, error: "" });
      return;
    }
    try {
      const list = await api.status(current);
      const nextGroups = Array.isArray(list) ? list : [];
      const selectedGroup = nextGroups.find((group) => group.name === current.group)
        || nextGroups.find((group) => String(group.type).toLowerCase() === "selector")
        || nextGroups[0];
      const groupName = selectedGroup ? selectedGroup.name : current.group || "";
      const group = nextGroups.find((item) => item.name === groupName);
      setGroups(nextGroups);
      setStatus({
        connected: true,
        current: group ? group.now || "" : "",
        groupName,
        members: group ? group.members || 0 : 0,
        error: "",
      });
      if (groupName && groupName !== current.group) updateSettings({ group: groupName });
    } catch (error) {
      setGroups([]);
      setStatus({ connected: false, current: "", groupName: current.group || "", members: 0, error: errText(error) });
      notify(errText(error), "error");
    }
  }, [notify]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [v, bootstrap, running, history, saved] = await Promise.all([
          api.version().catch(() => ""),
          api.bootstrap(),
          api.daemonStatus().catch(() => false),
          api.history().catch(() => []),
          api.results().catch(() => null),
        ]);
        if (cancelled) return;
        setVersion(v || "");
        setDaemon(!!running);
        setControllers(bootstrap.controllers || []);
        setSubscriptions(bootstrap.subscriptions || []);
        const normalized = mapSettings(bootstrap.settings || {});
        setSettings(normalized);
        settingsRef.current = normalized;
        setEvents(sortEvents(history));
        if (saved && Array.isArray(saved.results) && saved.results.length) {
          setResults(saved.results);
          setBest(saved.results[0] || null);
          setSelected(saved.results[0]?.name || null);
          if (!readLastTest() && saved.generated_at) {
            writeLastTest(saved.generated_at);
            setLastTestAt(saved.generated_at);
          }
        }
        hydratedRef.current = true;
        await refreshStatus(normalized);
      } catch (error) {
        if (!cancelled) notify(errText(error), "error");
      }
    })();

    const unsubscribe = subscribeDaemon((event) => {
      setEvents((previous) => {
        const key = `${event.time}|${event.kind}|${event.from}|${event.to}`;
        if (previous.some((item) => `${item.time}|${item.kind}|${item.from}|${item.to}` === key)) return previous;
        return [event, ...previous].slice(0, 50);
      });
    });

    return () => {
      cancelled = true;
      unsubscribe();
    };
  }, [notify, refreshStatus]);

  useEffect(() => {
    if (!hydratedRef.current) return undefined;
    clearTimeout(saveTimer.current);
    saveTimer.current = setTimeout(() => {
      api.saveSettings(settings).catch((error) => notify(`保存失败：${errText(error)}`, "error"));
    }, 500);
    return () => clearTimeout(saveTimer.current);
  }, [settings, notify]);

  useEffect(() => {
    const meta = VIEW_META[view] || VIEW_META.dashboard;
    document.title = `NodePulse · ${meta[0]}`;
    if (contentRef.current) contentRef.current.scrollTop = 0;
    setScrolled(false);
  }, [view]);

  useEffect(() => {
    if (!busy) {
      setElapsed(0);
      return undefined;
    }
    const start = Date.now();
    const timer = setInterval(() => setElapsed(Math.floor((Date.now() - start) / 1000)), 1000);
    return () => clearInterval(timer);
  }, [busy]);

  useEffect(() => {
    const timer = setInterval(() => setClockTick((value) => value + 1), 30000);
    return () => clearInterval(timer);
  }, []);

  function startBusy(kind) {
    setBusyKind(kind);
    setBusy(true);
  }

  function finishBusy() {
    setBusy(false);
    setBusyKind("");
  }

  function markTest() {
    const now = new Date().toISOString();
    setLastTestAt(now);
    writeLastTest(now);
  }

  async function runOneClick() {
    if (busy) return;
    const current = settingsRef.current;
    if (!current.api && !current.configPath) {
      notify("请先连接控制口，或选择一个订阅文件", "error");
      return;
    }
    startBusy("oneclick");
    try {
      const result = await api.oneClick(current, true);
      setResults(result.results || []);
      setBest(result.best || null);
      if (result.best) setSelected(result.best.name);
      markTest();
      await refreshStatus(current);
      notify(result.message || (result.best ? `推荐 ${result.best.name}` : "优化完成"), "info");
    } catch (error) {
      notify(errText(error), "error");
    } finally {
      finishBusy();
    }
  }

  async function runTest(goNodes = false) {
    if (busy) return;
    const current = settingsRef.current;
    if (!current.api && !current.configPath) {
      notify("请先连接控制口，或选择一个订阅文件", "error");
      return;
    }
    startBusy("test");
    try {
      const response = await api.test(current);
      const list = Array.isArray(response) ? response : [];
      setResults(list);
      const top = list[0] || null;
      setBest(top);
      setSelected(top ? top.name : null);
      markTest();
      await refreshStatus(current);
      if (goNodes) setView("nodes");
      notify(`测速完成：${list.length} 个节点`, "info");
    } catch (error) {
      notify(errText(error), "error");
    } finally {
      finishBusy();
    }
  }

  async function saveSettings() {
    if (busy) return;
    startBusy("save");
    try {
      await api.saveSettings(settingsRef.current);
      notify("设置已保存", "info");
    } catch (error) {
      notify(errText(error), "error");
    } finally {
      finishBusy();
    }
  }

  async function importSubscription() {
    const url = subURL.trim();
    if (!url) {
      notify("请粘贴订阅 URL", "error");
      return;
    }
    if (busy) return;
    startBusy("import");
    try {
      const path = await api.importSubscription(url);
      updateSettings({ configPath: path });
      setSubURL("");
      const bootstrap = await api.bootstrap();
      setControllers(bootstrap.controllers || []);
      setSubscriptions(bootstrap.subscriptions || []);
      notify("订阅已导入", "info");
    } catch (error) {
      notify(errText(error), "error");
    } finally {
      finishBusy();
    }
  }

  async function switchSelected() {
    if (!selected) {
      notify("请先在结果里选择一个节点", "error");
      return;
    }
    const current = settingsRef.current;
    if (!current.group) {
      notify("请先在设置里选择 Selector 组", "error");
      setView("settings");
      return;
    }
    if (busy) return;
    startBusy("switch");
    try {
      await api.switchNode(current, selected);
      await refreshStatus(current);
      notify(`已切换到 ${selected}`, "info");
    } catch (error) {
      notify(errText(error), "error");
    } finally {
      finishBusy();
    }
  }

  async function exportConfig() {
    const current = settingsRef.current;
    if (!current.configPath) {
      notify("请先选择订阅配置", "error");
      return;
    }
    if (busy) return;
    startBusy("export");
    try {
      const message = await api.exportConfig("stable.yaml", 0.8);
      notify(message || "已导出 stable.yaml", "info");
    } catch (error) {
      notify(errText(error), "error");
    } finally {
      finishBusy();
    }
  }

  async function toggleDaemon(enabled) {
    const current = settingsRef.current;
    if (enabled && !current.api) {
      notify("自动优化需要先连接客户端", "error");
      return;
    }
    try {
      if (enabled) await api.startDaemon(current, daemonInterval);
      else await api.stopDaemon();
      setDaemon(enabled);
      notify(enabled ? "后台自动优化已开启" : "已停止后台自动优化", "info");
    } catch (error) {
      setDaemon(!enabled);
      notify(errText(error), "error");
    }
  }

  function changeMode(mode) {
    const next = setMode(mode);
    applySeed(seedRef.current);
    setThemeMode(next);
  }

  function changeSeed(value) {
    const normalized = normalizeSeed(value);
    setSeed(normalized);
    seedRef.current = normalized;
    clearTimeout(seedTimer.current);
    seedTimer.current = setTimeout(() => applySeed(normalized), 60);
  }

  function resetSeed() {
    clearTimeout(seedTimer.current);
    applySeed(DEFAULT_SEED);
    setSeed(DEFAULT_SEED);
    seedRef.current = DEFAULT_SEED;
  }

  function toggleTheme() {
    changeMode(nextMode(themeMode));
  }

  function navigate(nextView) {
    setView(nextView);
  }

  const meta = VIEW_META[view] || VIEW_META.dashboard;
  const themeIcon = themeMode === "light" ? "sun" : themeMode === "dark" ? "moon" : "monitor";
  const themeTitle = `主题：${modeLabel(themeMode)}，点击切换`;
  const connClass = status.error ? "error" : status.connected ? "ok" : "warn";
  const connLabel = status.error ? "连接失败" : status.connected ? (controllers.length ? `已连接 ${controllers.length} 个客户端` : "已连接") : controllers.length ? "未连接" : "未检测到客户端";
  const lastTestLabel = lastTestAt ? timeAgo(lastTestAt) : "";
  const canSwitch = !!selected;
  const canExport = !!settings.configPath && results.length > 0;

  return html`<div class="app">
    <a class="skip-link" href="#content">跳到内容</a>
    <aside class="rail">
      <nav class="rail-nav" aria-label="主导航">
        ${NAV.map((item) => html`<button
          type="button"
          class=${"rail-item" + (view === item.view ? " active" : "")}
          data-view=${item.view}
          title=${item.label}
          aria-label=${item.label}
          aria-current=${view === item.view ? "page" : undefined}
          onClick=${() => navigate(item.view)}
        ><${Icon} name=${item.icon} /></button>`)}
      </nav>
    </aside>

    <main class="main">
      <header class=${"topbar" + (scrolled ? " scrolled" : "")}>
        <div class="topbar-title">
          <h1 aria-live="polite">${meta[0]}</h1>
          <p class="topbar-sub">${meta[1]}</p>
        </div>
        <div class="topbar-actions">
          <${IconButton} icon=${themeIcon} label=${themeTitle} title=${themeTitle} onClick=${toggleTheme} />
          <span class=${"status-chip " + connClass} role="status" aria-live="polite">${connLabel}</span>
          ${version ? html`<span class="version">v${version}</span>` : null}
        </div>
      </header>

      <div class="content" id="content" ref=${contentRef} tabindex="-1" onScroll=${(event) => setScrolled(event.currentTarget.scrollTop > 4)}>
        ${view === "dashboard"
          ? html`<${Dashboard}
              settings=${settings}
              status=${status}
              results=${results}
              best=${best}
              selected=${selected}
              busy=${busy}
              busyKind=${busyKind}
              elapsed=${elapsed}
              lastTestAt=${lastTestLabel}
              controllers=${controllers}
              subscriptions=${subscriptions}
              daemon=${daemon}
              daemonInterval=${daemonInterval}
              groups=${groups}
              subURL=${subURL}
              onOneClick=${runOneClick}
              onTest=${() => runTest(false)}
              onSelectNode=${setSelected}
              onRefresh=${() => refreshStatus()}
              onController=${(url) => { updateSettings({ api: url }); refreshStatus({ ...settingsRef.current, api: url }); }}
              onSettings=${updateSettings}
              onRounds=${(rounds) => updateSettings({ rounds })}
              onDaemon=${toggleDaemon}
              onInterval=${setDaemonInterval}
              onSubURL=${setSubURL}
              onImport=${importSubscription}
            />`
          : view === "nodes"
            ? html`<${Nodes}
                results=${results}
                best=${best}
                selected=${selected}
                onSelect=${setSelected}
                onRefresh=${() => refreshStatus()}
                onSwitch=${switchSelected}
                canSwitch=${canSwitch}
                busy=${busy && busyKind === "test"}
              />`
            : view === "settings"
              ? html`<${Settings}
                  settings=${settings}
                  groups=${groups}
                  seed=${seed}
                  mode=${themeMode}
                  busy=${busy}
                  busyKind=${busyKind}
                  lastTestAt=${lastTestLabel}
                  onSettings=${updateSettings}
                  onSeed=${changeSeed}
                  onResetSeed=${resetSeed}
                  onMode=${changeMode}
                  onSave=${saveSettings}
                  onTest=${() => runTest(true)}
                  onSwitch=${switchSelected}
                  onExport=${exportConfig}
                  canSwitch=${canSwitch}
                  canExport=${canExport}
                />`
              : html`<${History} events=${events} />`}
      </div>
    </main>
  </div>

  <nav class="bottom-nav" aria-label="底部导航">
    ${NAV.map((item) => html`<button
      type="button"
      class=${"bottom-item" + (view === item.view ? " active" : "")}
      data-view=${item.view}
      aria-current=${view === item.view ? "page" : undefined}
      onClick=${() => navigate(item.view)}
    ><${Icon} name=${item.icon} /><span>${item.label}</span></button>`)}
  </nav>

  <${Snackbar} snack=${snack} onClose=${() => setSnack(null)} />`;
}

bootTheme();

render(html`<${App} />`, document.getElementById("app"));
