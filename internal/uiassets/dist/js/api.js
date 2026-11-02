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
    return post("/api/test", {
      api: s.api,
      secret: s.secret,
      config_path: s.configPath,
      mihomo_path: s.mihomoPath,
      url: s.url,
      rounds: s.rounds,
      concurrency: s.concurrency,
      timeout_ms: s.timeoutMs,
    });
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

  async exportConfig(out, minSuccess) {
    if (wails()) return wails().Export(out, minSuccess);
    const data = await post("/api/export", { out, min_success: minSuccess });
    return (data && data.message) || "已导出稳定配置";
  },

  async importSubscription(url) {
    if (wails()) return wails().ImportSubscription(url);
    const data = await post("/api/import", { url });
    return data.path;
  },

  async startDaemon(s, intervalSec) {
    if (wails()) {
      return wails().StartDaemon(s.api, s.secret, s.group, intervalSec, s.rounds, 0.2, 2, 0.8);
    }
    return post("/api/daemon/start", {
      api: s.api,
      secret: s.secret,
      group: s.group,
      interval_sec: intervalSec,
      rounds: s.rounds,
      margin: 0.2,
      confirm: 2,
      min_success: 0.8,
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
