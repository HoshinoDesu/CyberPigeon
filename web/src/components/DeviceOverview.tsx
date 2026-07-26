"use client";

import { getSignalColor } from "@/lib/format";
import type { ModemInfo } from "@/lib/types";
import { SignalBars } from "./SignalBars";

type Props = {
  modems: ModemInfo[];
  loading?: boolean;
  compact?: boolean;
  onOpenDevices?: () => void;
};

export function DeviceOverview({
  modems,
  loading,
  compact,
  onOpenDevices,
}: Props) {
  const primary = modems[0];
  const extra = Math.max(0, modems.length - 1);

  if (loading && !primary) {
    return (
      <div className={`device-overview ${compact ? "device-overview-compact" : ""}`}>
        <div className="caption">设备速览</div>
        <div className="mt-2 flex items-center gap-2">
          <div className="spinner !h-3.5 !w-3.5 !border-[1.5px]" />
          <span className="footnote">读取中…</span>
        </div>
      </div>
    );
  }

  if (!primary) {
    return (
      <button
        type="button"
        className={`device-overview device-overview-btn ${compact ? "device-overview-compact" : ""}`}
        onClick={onOpenDevices}
      >
        <div className="caption">设备速览</div>
        <div className="mt-1.5 text-[0.8125rem] font-medium text-[var(--label-secondary)]">
          暂无在线设备
        </div>
      </button>
    );
  }

  const name = primary.model || primary.manufacturer || "未知基带";
  const carrier = primary.operator_name || "未知运营商";
  const q = primary.signal_quality ?? 0;

  return (
    <button
      type="button"
      className={`device-overview device-overview-btn ${compact ? "device-overview-compact" : ""}`}
      onClick={onOpenDevices}
    >
      <div className="mb-2 flex items-center justify-between gap-2">
        <span className="caption">设备速览</span>
        {extra > 0 && (
          <span className="caption mono">+{extra}</span>
        )}
      </div>

      <div className="min-w-0 text-left">
        <div className="truncate-safe text-[0.875rem] font-semibold tracking-[-0.015em] text-[var(--label)]">
          {name}
        </div>
        <div className="meta-value truncate-safe mt-0.5">
          {carrier}
          {primary.number ? ` · ${primary.number}` : ""}
        </div>
      </div>

      <div className="mt-2.5 flex items-center gap-2">
        <SignalBars quality={q} />
        <span
          className="mono shrink-0 text-[0.75rem] font-semibold tabular-nums"
          style={{ color: getSignalColor(q) }}
        >
          {q}%
        </span>
      </div>
    </button>
  );
}
