import { html } from "./html.js";
import { render } from "../vendor/preact.module.js";
import { useCallback, useEffect, useRef, useState } from "../vendor/hooks.module.js";
import { api, isWails } from "./api.js";
import { subscribeDaemon } from "./events.js";
import {
  currentMode,
  currentSeed,
  applySeed,
  bootTheme,
  normalizeSeed,
  setMode,
  DEFAULT_SEED,
} from "./theme.js";
import { Icon, ProfileMenu, Snackbar } from "./components.js";
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
  dashboard: ["仪表盘"],
  nodes: ["节点"],
  settings: ["设置"],
  history: ["历史"],
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
    profiles: [],
    activeProfile: "",
    autoOptimize: false,
    daemonInterval: 300,
    autoMargin: 0.2,
    autoConfirm: 2,
    autoMinSuccess: 0.8,
    closeToTray: true,
    notifyOnSwitch: true,
  };
}

function profileFields(source) {
  return {
    api: source.api || "",
    secret: source.secret || "",
    group: source.group || "",
    configPath: source.configPath || "",
    mihomoPath: source.mihomoPath || "",
    url: source.url || "https://www.gstatic.com/generate_204",
    rounds: Number(source.rounds) || 5,
    concurrency: Number(source.concurrency) || 8,
    timeoutMs: Number(source.timeoutMs) || 5000,
  };
}

function mapProfile(raw) {
  return {
    id: raw.id || "",
    name: raw.name || "未命名环境",
    ...profileFields({
      api: raw.api,
      secret: raw.secret,
      group: raw.group,
      configPath: raw.config_path,
      mihomoPath: raw.mihomo_path,
      url: raw.url,
      rounds: raw.rounds,
      concurrency: raw.concurrency,
      timeoutMs: raw.timeout_ms,
    }),
  };
}

function mapSettings(raw) {
  const base = profileFields({
    api: raw.api,
    secret: raw.secret,
    group: raw.group,
    configPath: raw.config_path,
    mihomoPath: raw.mihomo_path,
    url: raw.url,
    rounds: raw.rounds,
    concurrency: raw.concurrency,
    timeoutMs: raw.timeout_ms,
  });
  const profiles = Array.isArray(raw.profiles) ? raw.profiles.map(mapProfile) : [];
  if (profiles.length === 0) {
    profiles.push({ id: "default", name: "默认环境", ...base });
  }
  const activeProfile = profiles.some((profile) => profile.id === raw.active_profile)
    ? raw.active_profile
    : profiles[0].id;
  return {
    ...base,
    profiles,
    activeProfile,
    autoOptimize: !!raw.auto_optimize,
    daemonInterval: Number(raw.daemon_interval) || 300,
    autoMargin: Number(raw.auto_margin) || 0.2,
    autoConfirm: Number(raw.auto_confirm) || 2,
    autoMinSuccess: Number(raw.auto_min_success) || 0.8,
    closeToTray: raw.close_to_tray !== false,
    notifyOnSwitch: raw.notify_on_switch !== false,
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

function syncWindowTheme() {
  const dark = document.documentElement.dataset.theme === "dark";
  api.setWindowTheme(dark).catch(() => {});
}

function App() {
  const [view, setView] = useState("dashboard");
  const [settings, setSettings] = useState(defaultSettings);
  const [controllers, setControllers] = useState([]);
  const [clients, setClients] = useState([]);
  const [subscriptions, setSubscriptions] = useState([]);
  const [status, setStatus] = useState({ connected: false, current: "", groupName: "", members: 0, error: "" });
  const [groups, setGroups] = useState([]);
  const [results, setResults] = useState([]);
  const [best, setBest] = useState(null);
  const [selected, setSelected] = useState(null);
  const [daemon, setDaemon] = useState(false);
  const [autostartEnabled, setAutostartEnabled] = useState(false);
  const [health, setHealth] = useState(null);
  const [healthBusy, setHealthBusy] = useState(false);
  const [events, setEvents] = useState([]);
  const [busy, setBusy] = useState(false);
  const [busyKind, setBusyKind] = useState("");
  const [progress, setProgress] = useState(null);
  const [exportImpact, setExportImpact] = useState(null);
  const [resultScope, setResultScope] = useState("group");
  const [downloadUrl, setDownloadUrl] = useState("");
  const [downloadResult, setDownloadResult] = useState(null);
  const [exportMinimal, setExportMinimal] = useState(false);
  const [elapsed, setElapsed] = useState(0);
  const [lastTestAt, setLastTestAt] = useState(readLastTest);
  const [snack, setSnack] = useState(null);
  const [subURL, setSubURL] = useState("");
  const [version, setVersion] = useState("");
  const [themeMode, setThemeMode] = useState(currentMode);
  const [seed, setSeed] = useState(currentSeed);
  const [scrolled, setScrolled] = useState(false);
  const [detecting, setDetecting] = useState(false);
  const [, setClockTick] = useState(0);

  const settingsRef = useRef(settings);
  const seedRef = useRef(seed);
  const hydratedRef = useRef(false);
  const saveTimer = useRef(null);
  const seedTimer = useRef(null);
  const snackTimer = useRef(null);
  const contentRef = useRef(null);
  const canceledRef = useRef(false);

  settingsRef.current = settings;
  seedRef.current = seed;

  const notify = useCallback((text, kind = "info") => {
    clearTimeout(snackTimer.current);
    setSnack({ text: String(text), kind, id: Date.now() });
    snackTimer.current = setTimeout(() => setSnack(null), kind === "error" ? 8000 : 5000);
  }, []);

  function updateSettings(patch) {
    const previous = settingsRef.current;
    let next = { ...previous, ...patch };
    if (!("activeProfile" in patch) && !("profiles" in patch) && next.profiles?.length) {
      const fields = profileFields(next);
      next = {
        ...next,
        profiles: next.profiles.map((profile) =>
          profile.id === next.activeProfile ? { ...profile, ...fields } : profile
        ),
      };
    }
    settingsRef.current = next;
    setSettings(next);
  }

  async function clearEnvironmentResults() {
    setResults([]);
    setBest(null);
    setSelected(null);
    setLastTestAt("");
    try { localStorage.removeItem(LAST_TEST_KEY); } catch (_) {}
    try { await api.clearResults(); } catch (_) {}
  }

  async function stopDaemonForSwitch() {
    if (!daemon) return true;
    try {
      await api.stopDaemon();
      setDaemon(false);
      return true;
    } catch (error) {
      notify(`请先停止自动优化再切换环境：${errText(error)}`, "error");
      return false;
    }
  }

  async function activateProfile(id) {
    if (busy) {
      notify("当前有任务在进行，完成后再切换环境", "error");
      return;
    }
    const profile = settingsRef.current.profiles.find((item) => item.id === id);
    if (!profile) return;
    if (!(await stopDaemonForSwitch())) return;
    await clearEnvironmentResults();
    updateSettings({ ...profileFields(profile), activeProfile: profile.id });
    const ok = await refreshStatus(settingsRef.current);
    if (ok && settingsRef.current.autoOptimize) {
      if (profile.api) {
        try {
          await api.startDaemon(settingsRef.current);
          setDaemon(true);
        } catch (error) {
          notify(`新环境自动优化启动失败：${errText(error)}`, "error");
        }
      } else {
        updateSettings({ autoOptimize: false });
        notify("新环境没有控制口，自动优化已关闭", "error");
      }
    }
    if (ok) notify(`已切换到环境：${profile.name}`, "info");
  }

  function saveCurrentProfile() {
    const current = settingsRef.current;
    const active = current.profiles.find((item) => item.id === current.activeProfile);
    if (!active) {
      notify("当前没有可保存的环境", "error");
      return;
    }
    const fields = profileFields(current);
    updateSettings({
      profiles: current.profiles.map((profile) =>
        profile.id === active.id ? { ...profile, ...fields, name: active.name } : profile
      ),
    });
    notify(`已保存到环境：${active.name}`, "info");
  }

  function newProfile() {
    const current = settingsRef.current;
    const profile = {
      id: `env-${Date.now().toString(36)}`,
      name: `环境 ${current.profiles.length + 1}`,
      ...profileFields(current),
    };
    updateSettings({ profiles: [...current.profiles, profile], activeProfile: profile.id });
    notify(`已新建环境：${profile.name}`, "info");
  }

  function renameProfile(name) {
    const current = settingsRef.current;
    updateSettings({
      profiles: current.profiles.map((profile) =>
        profile.id === current.activeProfile ? { ...profile, name: String(name || "") } : profile
      ),
    });
  }

  function commitProfileName(name) {
    if (!String(name || "").trim()) renameProfile("未命名环境");
  }

  async function deleteProfile() {
    if (busy) {
      notify("当前有任务在进行，完成后再删除环境", "error");
      return;
    }
    const current = settingsRef.current;
    if (current.profiles.length <= 1) {
      notify("至少保留一个环境", "error");
      return;
    }
    if (!(await stopDaemonForSwitch())) return;
    const remaining = current.profiles.filter((profile) => profile.id !== current.activeProfile);
    const next = remaining[0];
    await clearEnvironmentResults();
    updateSettings({ profiles: remaining, activeProfile: next.id, ...profileFields(next) });
    const ok = await refreshStatus(settingsRef.current);
    if (ok && settingsRef.current.autoOptimize) {
      if (next.api) {
        try {
          await api.startDaemon(settingsRef.current);
          setDaemon(true);
        } catch (error) {
          notify(`自动优化启动失败：${errText(error)}`, "error");
        }
      } else {
        updateSettings({ autoOptimize: false });
      }
    }
    if (ok) notify(`已删除环境，切换到：${next.name}`, "info");
  }

  const refreshStatus = useCallback(async (override) => {
    const current = override || settingsRef.current;
    if (!current.api) {
      setGroups([]);
      setStatus({ connected: false, current: "", groupName: current.group || "", members: 0, error: "" });
      return true;
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
      return true;
    } catch (error) {
      setGroups([]);
      setStatus({ connected: false, current: "", groupName: current.group || "", members: 0, error: errText(error) });
      notify(errText(error), "error");
      return false;
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
        setAutostartEnabled(!!bootstrap.autostart);
        setControllers(bootstrap.controllers || []);
        setClients(bootstrap.clients || []);
        setSubscriptions(bootstrap.subscriptions || []);
        const normalized = mapSettings(bootstrap.settings || {});
        setSettings(normalized);
        settingsRef.current = normalized;
        setEvents(sortEvents(history));
        const sameEnvironment = !saved?.config_path || saved.config_path === normalized.configPath;
        if (saved && Array.isArray(saved.results) && saved.results.length && sameEnvironment) {
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
        refreshHealth();
        if (normalized.autoOptimize && normalized.api && !running) {
          try {
            await api.startDaemon(normalized);
            setDaemon(true);
            notify("后台自动优化已恢复", "info");
          } catch (error) {
            notify(`自动优化恢复失败：${errText(error)}`, "error");
          }
        }
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
    if (!isWails() || !window.runtime || !window.runtime.EventsOn) return undefined;
    window.runtime.EventsOn("nodepulse:autostart", setAutostartEnabled);
    return () => {
      if (window.runtime.EventsOff) window.runtime.EventsOff("nodepulse:autostart");
    };
  }, []);

  useEffect(() => {
    if (!busy || !["oneclick", "test", "all", "download"].includes(busyKind)) {
      setProgress(null);
      return undefined;
    }
    let stopped = false;
    const tick = async () => {
      try {
        const snapshot = await api.progress();
        if (!stopped) setProgress(snapshot);
      } catch (_) {}
    };
    tick();
    const timer = setInterval(tick, 500);
    return () => {
      stopped = true;
      clearInterval(timer);
    };
  }, [busy, busyKind]);

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

  function clearRunState() {
    setResults([]);
    setBest(null);
    setSelected(null);
    setExportImpact(null);
    setLastTestAt("");
    try { localStorage.removeItem(LAST_TEST_KEY); } catch (_) {}
  }

  async function runOneClick() {
    if (busy) return;
    if (daemon) {
      notify("后台自动优化运行中，请先暂停再手动测速", "error");
      return;
    }
    canceledRef.current = false;
    const current = settingsRef.current;
    if (!current.api && !current.configPath) {
      notify("请先连接控制口，或选择一个订阅文件", "error");
      return;
    }
    clearRunState();
    startBusy("oneclick");
    try {
      const result = await api.oneClick(current, true);
      setResults(result.results || []);
      setBest(result.best || null);
      setResultScope("group");
      if (result.best) setSelected(result.best.name);
      markTest();
      const ok = await refreshStatus(current);
      if (ok) notify(result.message || (result.best ? `推荐 ${result.best.name}` : "优化完成"), "info");
    } catch (error) {
      if (canceledRef.current) notify("已取消测速", "info");
      else notify(errText(error), "error");
      api.clearResults().catch(() => {});
    } finally {
      finishBusy();
    }
  }

  async function runTest(goNodes = false) {
    if (busy) return;
    if (daemon) {
      notify("后台自动优化运行中，请先暂停再手动测速", "error");
      return;
    }
    canceledRef.current = false;
    const current = settingsRef.current;
    if (!current.api && !current.configPath) {
      notify("请先连接控制口，或选择一个订阅文件", "error");
      return;
    }
    clearRunState();
    startBusy("test");
    try {
      const response = await api.test(current);
      const list = Array.isArray(response) ? response : [];
      setResults(list);
      const top = list[0] || null;
      setBest(top);
      setResultScope("group");
      setSelected(top ? top.name : null);
      markTest();
      const ok = await refreshStatus(current);
      if (goNodes) setView("nodes");
      if (ok) notify(`测速完成：${list.length} 个节点`, "info");
    } catch (error) {
      if (canceledRef.current) notify("已取消测速", "info");
      else notify(errText(error), "error");
      api.clearResults().catch(() => {});
    } finally {
      finishBusy();
    }
  }

  async function runAllSubscriptions() {
    if (busy) return;
    if (daemon) {
      notify("后台自动优化运行中，请先暂停再联合测速", "error");
      return;
    }
    canceledRef.current = false;
    clearRunState();
    startBusy("all");
    try {
      const result = await api.testAllSubscriptions(settingsRef.current);
      const list = result?.results || [];
      setResults(list);
      setBest(result?.best || null);
      setSelected(null);
      setResultScope("all");
      markTest();
      await refreshStatus(settingsRef.current);
      setView("nodes");
      notify(result?.message || `联合测速完成：${list.length} 个节点`, "info");
    } catch (error) {
      if (canceledRef.current) notify("已取消测速", "info");
      else notify(errText(error), "error");
      api.clearResults().catch(() => {});
    } finally {
      finishBusy();
    }
  }

  async function runDownloadMode() {
    if (busy) return;
    if (daemon) {
      notify("后台自动优化运行中，请先暂停再测下载模式", "error");
      return;
    }
    const target = downloadUrl.trim();
    if (!/^https?:\/\//i.test(target)) {
      notify("请填写可直接访问的 http/https 文件 URL", "error");
      return;
    }
    canceledRef.current = false;
    clearRunState();
    startBusy("download");
    try {
      const result = await api.downloadMode(settingsRef.current, target);
      setDownloadResult(result);
      const list = result?.results || [];
      setResults(list);
      setBest(result?.best || null);
      setSelected(result?.best?.name || null);
      setResultScope("download");
      markTest();
      await refreshStatus(settingsRef.current);
      const failed = result?.reason_code === "download-unreachable";
      notify(result?.message || (failed ? "目标站点在当前节点均不可达" : "下载模式完成"), failed ? "error" : "info");
    } catch (error) {
      if (canceledRef.current) notify("已取消测速", "info");
      else notify(errText(error), "error");
      api.clearResults().catch(() => {});
    } finally {
      finishBusy();
    }
  }

  async function cancelCurrentTest() {
    try {
      const canceled = await api.cancelTest();
      if (canceled === false) {
        notify("没有正在运行的测速", "error");
        return;
      }
      canceledRef.current = true;
      notify("正在取消测速…", "info");
    } catch (error) {
      notify(errText(error), "error");
    }
  }

  async function backupSettings() {
    if (busy) return;
    startBusy("backup");
    try {
      const result = await api.exportSettingsBackup();
      notify(result && result.includes("/") ? `已导出：${result}` : (result || "已导出设置备份"), "info");
    } catch (error) {
      notify(errText(error), "error");
    } finally {
      finishBusy();
    }
  }

  async function restoreSettings(rawText) {
    if (busy) return;
    startBusy("restore");
    try {
      const message = await api.restoreSettings(rawText);
      notify(message || "设置已恢复，正在重新加载…", "info");
      setTimeout(() => window.location.reload(), 900);
    } catch (error) {
      notify(errText(error), "error");
    } finally {
      finishBusy();
    }
  }

  async function detectClients() {
    if (detecting) return;
    setDetecting(true);
    try {
      const bootstrap = await api.bootstrap();
      const nextClients = bootstrap.clients || [];
      setControllers(bootstrap.controllers || []);
      setClients(nextClients);
      setSubscriptions(bootstrap.subscriptions || []);
      const current = settingsRef.current;
      const patch = {};
      const found = bootstrap.settings || {};
      if (!current.api && found.api) patch.api = found.api;
      if (!current.secret && found.secret) patch.secret = found.secret;
      if (!current.configPath && found.config_path) patch.configPath = found.config_path;
      if (Object.keys(patch).length) updateSettings(patch);
      const ok = await refreshStatus({ ...current, ...patch });
      const running = nextClients.filter((client) => client.running);
      notify(
        running.length
          ? `检测到 ${running.map((client) => client.name).join("、")}${ok ? "" : "，但控制口未连接"}`
          : "没有检测到运行中的客户端",
        running.length ? "info" : "error"
      );
    } catch (error) {
      notify(errText(error), "error");
    } finally {
      setDetecting(false);
    }
  }

  async function refreshHealth() {
    if (healthBusy) return;
    setHealthBusy(true);
    try {
      setHealth(await api.health(settingsRef.current));
    } catch (error) {
      notify(errText(error), "error");
    } finally {
      setHealthBusy(false);
    }
  }

  async function toggleAutostart(enabled) {
    try {
      await api.setAutostart(enabled);
      setAutostartEnabled(enabled);
      notify(enabled ? "已开启开机启动（隐藏到托盘）" : "已关闭开机启动", "info");
    } catch (error) {
      notify(errText(error), "error");
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
      notify("订阅已导入", "info");
      try {
        const bootstrap = await api.bootstrap();
        setControllers(bootstrap.controllers || []);
        setClients(bootstrap.clients || []);
        setSubscriptions(bootstrap.subscriptions || []);
      } catch (_) {}
    } catch (error) {
      notify(errText(error), "error");
    } finally {
      finishBusy();
    }
  }

  async function switchSelected() {
    if (resultScope !== "group" || results.some((row) => row.source)) {
      notify("联合测速只给推荐；请在客户端切换到对应订阅后再操作", "error");
      return;
    }
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
      const ok = await refreshStatus(current);
      if (ok) notify(`已切换到 ${selected}`, "info");
    } catch (error) {
      notify(errText(error), "error");
    } finally {
      finishBusy();
    }
  }

  async function previewExport(minimal = false) {
    const current = settingsRef.current;
    if (!current.configPath) {
      notify("请先选择订阅配置", "error");
      return;
    }
    if (busy) return;
    setExportMinimal(!!minimal);
    startBusy("export");
    try {
      const impact = await api.exportPreview(0.8, minimal);
      setExportImpact(impact);
    } catch (error) {
      notify(errText(error), "error");
    } finally {
      finishBusy();
    }
  }

  async function confirmExport() {
    if (!exportImpact || busy) return;
    startBusy("export");
    try {
      const message = await api.exportConfig("stable.yaml", 0.8, exportMinimal);
      notify(message || "已导出 stable.yaml", "info");
      setExportImpact(null);
    } catch (error) {
      notify(errText(error), "error");
    } finally {
      finishBusy();
    }
  }

  function cancelExport() {
    setExportImpact(null);
  }

  async function toggleDaemon(enabled) {
    if (busy) {
      notify("当前有测速任务在进行，完成后再切换自动优化", "error");
      return;
    }
    const current = settingsRef.current;
    if (enabled && !current.api) {
      notify("自动优化需要先连接客户端", "error");
      return;
    }
    try {
      if (enabled) await api.startDaemon(current);
      else await api.stopDaemon();
      setDaemon(enabled);
      updateSettings({ autoOptimize: enabled });
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
    syncWindowTheme();
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

  function navigate(nextView) {
    setView(nextView);
  }

  const meta = VIEW_META[view] || VIEW_META.dashboard;
  const connClass = status.error ? "error" : status.connected ? "ok" : "warn";
  const runningClients = clients.filter((client) => client.running);
  const connLabel = status.error
    ? "连接失败"
    : status.connected
      ? "已连接"
      : runningClients.length
        ? `检测到 ${runningClients[0].name}`
        : "未检测到客户端";
  const lastTestLabel = lastTestAt ? timeAgo(lastTestAt) : "";
  const latestEvent = events.find((event) =>
    ["cycle", "switch", "rollback", "error"].includes(event.kind)
    && (!event.group || !status.groupName || event.group === status.groupName)
  ) || null;
  const canSwitch = !!selected && resultScope === "group" && !results.some((row) => row.source);
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
        </div>
        <div class="topbar-actions">
          <${ProfileMenu}
            profiles=${settings.profiles}
            activeId=${settings.activeProfile}
            onSelect=${activateProfile}
            onManage=${() => navigate("settings")}
          />
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
              progress=${progress}
              onCancel=${cancelCurrentTest}
              lastTestAt=${lastTestLabel}
              controllers=${controllers}
              clients=${clients}
              detecting=${detecting}
              onDetect=${detectClients}
              subscriptions=${subscriptions}
              daemon=${daemon}
              daemonInterval=${settings.daemonInterval}
              latestEvent=${latestEvent}
              groups=${groups}
              subURL=${subURL}
              onOneClick=${runOneClick}
              onTest=${() => runTest(false)}
              onTestAll=${runAllSubscriptions}
              downloadUrl=${downloadUrl}
              onDownloadUrl=${setDownloadUrl}
              downloadResult=${downloadResult}
              onDownloadMode=${runDownloadMode}
              onSelectNode=${setSelected}
              onRefresh=${() => refreshStatus()}
              onController=${(url, secret) => {
                const patch = secret ? { api: url, secret } : { api: url };
                updateSettings(patch);
                refreshStatus({ ...settingsRef.current, ...patch });
              }}
              onSettings=${updateSettings}
              onRounds=${(rounds) => updateSettings({ rounds })}
              onDaemon=${toggleDaemon}
              onInterval=${(interval) => updateSettings({ daemonInterval: interval })}
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
                  progress=${progress}
                  onCancel=${cancelCurrentTest}
                  lastTestAt=${lastTestLabel}
                  onSettings=${updateSettings}
                  onSeed=${changeSeed}
                  onResetSeed=${resetSeed}
                  onMode=${changeMode}
                  onSave=${saveSettings}
                  onTest=${() => runTest(true)}
                  onSwitch=${switchSelected}
                  onExport=${previewExport}
                  onConfirmExport=${confirmExport}
                  onCancelExport=${cancelExport}
                  exportImpact=${exportImpact}
                  exportMinimal=${exportMinimal}
                  onExportMinimal=${(value) => previewExport(value)}
                  desktop=${isWails()}
                  autostart=${autostartEnabled}
                  onAutostart=${toggleAutostart}
                  onBackup=${backupSettings}
                  onRestore=${restoreSettings}
                  health=${health}
                  healthBusy=${healthBusy}
                  onHealth=${refreshHealth}
                  canSwitch=${canSwitch}
                  canExport=${canExport}
                  profiles=${settings.profiles}
                  activeProfile=${settings.activeProfile}
                  onActivateProfile=${activateProfile}
                  onRenameProfile=${renameProfile}
                  onProfileNameBlur=${commitProfileName}
                  onSaveProfile=${saveCurrentProfile}
                  onNewProfile=${newProfile}
                  onDeleteProfile=${deleteProfile}
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

bootTheme(syncWindowTheme);
syncWindowTheme();

render(html`<${App} />`, document.getElementById("app"));
