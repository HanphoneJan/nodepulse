// Single backend boundary. Wails bindings and the local HTTP server expose the
// same surface; no component talks to window.go or fetch directly.

const wails = () => window.go && window.go.main && window.go.main.App;

async function request(path, options) {
  const resp = await fetch(path, Object.assign({
    headers: { "Content-Type": "application/json" },
  }, options || {}));
  let data = null;
  try { data = await resp.json(); } catch (_) {}
  if (!resp.ok) {
    throw new Error((data && data.error) || resp.statusText || "请求失败");
  }
  return data;
}

const post = (path, body) => request(path, { method: "POST", body: JSON.stringify(body || {}) });

function canceledError() {
  const error = new Error("已取消测速");
  error.name = "CanceledError";
  return error;
}
const get = (path) => request(path);

function profilePayload(p) {
  return {
    id: p.id,
    name: p.name,
    api: p.api,
    secret: p.secret,
    group: p.group,
    config_path: p.configPath,
    mihomo_path: p.mihomoPath,
    url: p.url,
    rounds: p.rounds,
    concurrency: p.concurrency,
    timeout_ms: p.timeoutMs,
  };
}

function settingsPayload(s) {
  return {
    api: s.api,
    secret: s.secret,
    group: s.group,
    config_path: s.configPath,
    mihomo_path: s.mihomoPath,
    url: s.url,
    rounds: s.rounds,
    concurrency: s.concurrency,
    timeout_ms: s.timeoutMs,
    profiles: (s.profiles || []).map(profilePayload),
    active_profile: s.activeProfile,
    auto_optimize: !!s.autoOptimize,
    daemon_interval: s.daemonInterval,
    auto_margin: s.autoMargin,
    auto_confirm: s.autoConfirm,
    auto_min_success: s.autoMinSuccess,
    close_to_tray: !!s.closeToTray,
    notify_on_switch: !!s.notifyOnSwitch,
  };
}

export const api = {
  async version() {
    if (wails()) return wails().Version();
    const data = await get("/api/version");
    return data.version;
  },

  async bootstrap() {
    if (wails()) return wails().Bootstrap();
    return get("/api/bootstrap");
  },

  async health(s) {
    if (wails()) {
      return wails().Health(s.api, s.secret, s.group, s.configPath, s.mihomoPath, s.url);
    }
    return post("/api/health", {
      api: s.api,
      secret: s.secret,
      group: s.group,
      config_path: s.configPath,
      mihomo_path: s.mihomoPath,
      url: s.url,
    });
  },

  async saveSettings(s) {
    if (wails()) return wails().SaveSettings(settingsPayload(s));
    return post("/api/settings", settingsPayload(s));
  },

  async oneClick(s, doSwitch) {
    if (wails()) {
      return wails().OneClick(s.api, s.secret, s.group, s.configPath, s.mihomoPath, s.url, s.rounds, s.concurrency, s.timeoutMs, !!doSwitch);
    }
    return post("/api/oneclick", {
      api: s.api,
      secret: s.secret,
      group: s.group,
      config_path: s.configPath,
      mihomo_path: s.mihomoPath,
      url: s.url,
      rounds: s.rounds,
      concurrency: s.concurrency,
      timeout_ms: s.timeoutMs,
      switch: !!doSwitch,
    });
  },

  async test(s) {
    if (wails()) {
      return wails().Test(s.api, s.secret, s.configPath, s.mihomoPath, s.url, s.rounds, s.concurrency, s.timeoutMs);
    }
    const data = await post("/api/test", {
      api: s.api,
      secret: s.secret,
      config_path: s.configPath,
      mihomo_path: s.mihomoPath,
      url: s.url,
      rounds: s.rounds,
      concurrency: s.concurrency,
      timeout_ms: s.timeoutMs,
    });
    if (data && data.canceled) throw canceledError();
    return data;
  },

  async status(s) {
    if (wails()) return wails().Status(s.api, s.secret);
    // POST keeps the secret out of the URL and request logs.
    return post("/api/status", { api: s.api, secret: s.secret });
  },

  async switchNode(s, node) {
    if (wails()) return wails().Switch(s.api, s.secret, s.group, node);
    return post("/api/switch", { api: s.api, secret: s.secret, group: s.group, node });
  },

  async exportPreview(minSuccess, minimal) {
    if (wails()) return wails().PreviewExport(minSuccess, !!minimal);
    return post("/api/export/preview", { min_success: minSuccess, minimal: !!minimal });
  },

  async exportConfig(out, minSuccess, minimal) {
    if (wails()) return wails().Export(out, minSuccess, !!minimal);
    const data = await post("/api/export", { out, min_success: minSuccess, minimal: !!minimal });
    return (data && data.message) || "已导出稳定配置";
  },

  async importSubscription(url) {
    if (wails()) return wails().ImportSubscription(url);
    const data = await post("/api/import", { url });
    return data.path;
  },

  async exportSettingsBackup() {
    if (wails() && wails().ExportSettingsBackup) return wails().ExportSettingsBackup();
    const a = document.createElement("a");
    a.href = "/api/settings/backup";
    a.download = "nodepulse-settings.json";
    document.body.appendChild(a);
    a.click();
    a.remove();
    return "已下载设置备份";
  },

  async restoreSettings(rawText) {
    let parsed;
    try {
      parsed = JSON.parse(rawText);
    } catch (_) {
      throw new Error("备份文件不是有效的 JSON");
    }
    if (wails() && wails().RestoreSettings) {
      await wails().RestoreSettings(parsed);
      return "设置已恢复";
    }
    return post("/api/settings/restore", parsed);
  },

  async autostart() {
    if (wails() && wails().Autostart) return wails().Autostart();
    return false;
  },

  async setAutostart(enabled) {
    if (wails() && wails().SetAutostart) return wails().SetAutostart(!!enabled);
    throw new Error("只有桌面版支持开机启动");
  },

  async startDaemon(s) {
    if (wails()) {
      return wails().StartDaemon(s.api, s.secret, s.group, s.daemonInterval, s.rounds, s.autoMargin, s.autoConfirm, s.autoMinSuccess);
    }
    return post("/api/daemon/start", {
      api: s.api,
      secret: s.secret,
      group: s.group,
      interval_sec: s.daemonInterval,
      rounds: s.rounds,
      margin: s.autoMargin,
      confirm: s.autoConfirm,
      min_success: s.autoMinSuccess,
    });
  },

  async stopDaemon() {
    if (wails()) return wails().StopDaemon();
    return post("/api/daemon/stop", {});
  },

  async daemonStatus() {
    if (wails()) return wails().DaemonStatus();
    const data = await get("/api/daemon/status");
    return !!data.running;
  },

  async history() {
    if (wails() && wails().History) return wails().History();
    if (!wails()) return get("/api/history");
    return [];
  },

  async progress() {
    if (wails() && wails().Progress) return wails().Progress();
    if (!wails()) return get("/api/progress");
    return null;
  },

  async cancelTest() {
    if (wails() && wails().CancelTest) return wails().CancelTest();
    if (!wails()) {
      const data = await post("/api/cancel", {});
      return !!(data && data.canceled);
    }
    return false;
  },

  async clearResults() {
    if (wails() && wails().ClearResults) return wails().ClearResults();
    if (!wails()) return post("/api/results/clear", {});
  },

  async results() {
    if (wails()) {
      if (!wails().Results) return null;
      const list = await wails().Results();
      return Array.isArray(list) && list.length ? { results: list } : null;
    }
    try {
      return await get("/api/results");
    } catch (_) {
      return null;
    }
  },
};

export function isWails() {
  return !!wails();
}
