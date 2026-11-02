// One subscription point for daemon events, regardless of transport.

import { isWails } from "./api.js";

export function subscribeDaemon(onEvent) {
  if (isWails() && window.runtime && window.runtime.EventsOn) {
    window.runtime.EventsOn("nodepulse:daemon", onEvent);
    return () => {
      if (window.runtime.EventsOff) window.runtime.EventsOff("nodepulse:daemon");
    };
  }

  let timer = null;
  let last = "";
  let stopped = false;

  const poll = async () => {
    if (stopped || document.hidden) return;
    try {
      const resp = await fetch("/api/history", { headers: { Accept: "application/json" } });
      if (!resp.ok) return;
      const events = await resp.json();
      if (!Array.isArray(events) || !events.length) return;
      const newest = events[events.length - 1];
      const key = JSON.stringify(newest);
      if (key !== last) {
        last = key;
        onEvent(newest);
      }
    } catch (_) {}
  };

  const schedule = () => {
    clearInterval(timer);
    timer = setInterval(poll, document.hidden ? 30000 : 10000);
  };

  const onVisibility = () => {
    schedule();
    if (!document.hidden) poll();
  };

  document.addEventListener("visibilitychange", onVisibility);
  schedule();

  return () => {
    stopped = true;
    clearInterval(timer);
    document.removeEventListener("visibilitychange", onVisibility);
  };
}
