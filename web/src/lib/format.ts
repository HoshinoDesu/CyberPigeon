export function formatRelativeTime(timestamp?: string): string {
  if (!timestamp) return "未知";
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) return "未知";
  const diff = Date.now() - date.getTime();
  if (diff < 60_000) return "刚刚";
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)} 分钟前`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)} 小时前`;
  if (diff < 604_800_000) return `${Math.floor(diff / 86_400_000)} 天前`;
  return date.toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function formatClockTime(date = new Date()): string {
  return date.toLocaleTimeString("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function getSignalColor(q: number): string {
  if (q >= 80) return "var(--signal-excellent)";
  if (q >= 60) return "var(--signal-good)";
  if (q >= 40) return "var(--signal-fair)";
  if (q > 0) return "var(--signal-poor)";
  return "var(--signal-none)";
}
