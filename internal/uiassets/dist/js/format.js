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
  if (REASON_LABELS[reason]) return REASON_LABELS[reason];
  const map = {
    "current below min-success": "当前节点成功率低于阈值",
    "candidate better, waiting for confirmation": "发现更优节点，等待确认",
    "switch verified": "切换已验证",
    "rollback failed": "回滚失败",
    "context canceled": "已停止",
  };
  return map[reason] || reason || "";
}

export function hostOf(api) {
  if (!api) return "未连接客户端";
  try { return new URL(api).host; } catch (_) { return api; }
}

export function baseName(path) {
  if (!path) return "未选择订阅";
  const parts = String(path).split(/[\\/]/);
  return parts[parts.length - 1] || path;
}

const REASON_LABELS = {
  "controller-error": "控制口读取失败",
  "group-error": "管理组不可用",
  "no-candidates": "管理组没有可测节点",
  "no-qualified": "没有节点达到最低成功率",
  "baseline-failed": "本地基线失败（网络或测试 URL）",
  "all-nodes-failed": "所有节点未达标",
  "switch-failed": "切换失败",
  "verify-failed": "切换验证失败",
  "rollback-failed": "回滚失败",
  "no-current": "当前没有选中节点",
  "current-unhealthy": "当前节点不达标",
  "already-best": "当前已是最优",
  "better-candidate": "发现更优节点并切换",
  "better-candidate-confirming": "发现更优节点，等待确认",
  "within-margin": "候选优势不足，保持当前",
  "recommendation": "仅推荐，未切换",
  "switched": "已切换",
  "dry-run": "试运行，未切换",
};

export function confidenceLabel(confidence) {
  if (confidence === "high") return "高";
  if (confidence === "medium") return "中";
  if (confidence === "low") return "低";
  return "";
}

export function kindLabel(kind) {
  const map = { cycle: "检查", switch: "切换", rollback: "回滚", error: "错误" };
  return map[kind] || kind || "";
}
