export function timeAgo(iso) {
  if (!iso) return "";
  const then = new Date(iso).getTime();
  if (!Number.isFinite(then)) return "";
  const seconds = Math.max(0, Math.round((Date.now() - then) / 1000));
  if (seconds < 45) return "刚刚";
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes} 分钟前`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours} 小时前`;
  const days = Math.round(hours / 24);
  if (days < 7) return `${days} 天前`;
  return new Date(then).toLocaleDateString("zh-CN");
}

export function formatClock(iso) {
  if (!iso) return "";
  const date = new Date(iso);
  if (!Number.isFinite(date.getTime())) return "";
  return date.toLocaleTimeString("zh-CN", { hour12: false, hour: "2-digit", minute: "2-digit", second: "2-digit" });
}

export function latency(ms) {
  return Number.isFinite(ms) && ms > 0 ? `${Math.round(ms)}ms` : "—";
}

export function score(value) {
  return Number.isFinite(value) ? Number(value).toFixed(3) : "—";
}

export function percent(value) {
  return Number.isFinite(value) ? `${Math.round(value * 100)}%` : "—";
}

export function reasonLabel(reason) {
  const map = {
    "current below min-success": "当前节点成功率低于阈值",
    "candidate better, waiting for confirmation": "发现更优节点，等待确认",
    "switch verified": "切换已验证",
    "rollback failed": "回滚失败",
    "context canceled": "已停止",
  };
  return map[reason] || reason || "";
}
