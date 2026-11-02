// Web shim: when the UI is served by `nodepulse serve` instead of Wails,
// expose the same window.go.main.App surface backed by HTTP endpoints.
(function () {
  if (window.go && window.go.main && window.go.main.App) return;

  async function call(path, opts) {
    const resp = await fetch(path, Object.assign({
      headers: { "Content-Type": "application/json" },
    }, opts || {}));
    let data = null;
    try { data = await resp.json(); } catch (_) {}
    if (!resp.ok) throw new Error((data && data.error) || resp.statusText);
    return data;
  }
  const post = (path, body) => call(path, { method: "POST", body: JSON.stringify(body || {}) });
  const q = (v) => encodeURIComponent(v == null ? "" : v);

  window.go = {
    main: {
      App: {
        Version: async () => "0.3.0",
        Bootstrap: () => call("/api/bootstrap"),
        SaveSettings: (s) => post("/api/settings", s),
        OneClick: (api, secret, group, configPath, mihomoPath, rounds, doSwitch) =>
          post("/api/oneclick", {
            api, secret, group,
            config_path: configPath, mihomo_path: mihomoPath,
            rounds, switch: !!doSwitch,
          }),
        ImportSubscription: async (url) => (await post("/api/import", { url })).path,
        Status: (api, secret) => call(`/api/status?api=${q(api)}&secret=${q(secret)}`),
        Test: (api, secret, configPath, mihomoPath, url, rounds, concurrency, timeoutMs) =>
          post("/api/test", {
            api, secret, config_path: configPath, mihomo_path: mihomoPath,
            url, rounds, concurrency, timeout_ms: timeoutMs,
          }),
        Switch: (api, secret, group, node) => post("/api/switch", { api, secret, group, node }),
        Export: (out, minSuccess) => post("/api/export", { out, min_success: minSuccess }),
        StartDaemon: (api, secret, group, intervalSec, rounds, margin, confirm, minSuccess) =>
          post("/api/daemon/start", {
            api, secret, group, interval_sec: intervalSec, rounds,
            margin, confirm, min_success: minSuccess,
          }),
        StopDaemon: () => post("/api/daemon/stop", {}),
        DaemonStatus: async () => !!(await call("/api/daemon/status")).running,
      },
    },
  };

  window.runtime = window.runtime || {};
  if (!window.runtime.EventsOn) {
    let last = "";
    window.runtime.EventsOn = (name, cb) => {
      if (name !== "nodepulse:daemon") return;
      setInterval(async () => {
        try {
          const events = await call("/api/history");
          if (Array.isArray(events) && events.length) {
            const newest = events[events.length - 1];
            const key = JSON.stringify(newest);
            if (key !== last) { last = key; cb(newest); }
          }
        } catch (_) {}
      }, 5000);
    };
  }
})();
