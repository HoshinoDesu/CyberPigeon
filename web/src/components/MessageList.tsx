"use client";

import { useRef, useState } from "react";
import type { Message, ModemInfo } from "@/lib/types";
import { MessageCard } from "./MessageCard";
import { IconAlert, IconInboxLarge, IconSearch } from "./icons";

type Props = {
  messages: Message[];
  filtered: Message[];
  loading: boolean;
  error: string | null;
  searchQuery: string;
  onSearchChange: (v: string) => void;
  hasMore: boolean;
  loadingMore: boolean;
  onLoadMore: () => void;
  onDelete: (msg: Message) => void;
  onDeleteMany: (ids: string[]) => void;
  onRefresh: () => Promise<void>;
  total: number;
  modemNames: Record<string, string>;
  modems: ModemInfo[];
  deviceFilter: string;
  onDeviceFilterChange: (imei: string) => void;
};

function MessageSkeleton() {
  return (
    <div className="flex flex-col gap-3">
      {[0, 1, 2].map((i) => (
        <div key={i} className="glass-card p-4 sm:p-5" style={{ opacity: 1 - i * 0.25 }}>
          <div className="mb-3 flex items-start justify-between gap-3">
            <div className="min-w-0 flex-1">
              <div className="skeleton h-5 w-32" />
              <div className="skeleton mt-2 h-3.5 w-44" />
            </div>
            <div className="skeleton h-3.5 w-14" />
          </div>
          <div className="skeleton h-16 w-full !rounded-[var(--radius-md)]" />
        </div>
      ))}
    </div>
  );
}

const PTR_THRESHOLD = 64;

function rubberband(d: number, dim = 200, c = 0.5): number {
  return (d * dim * c) / (dim + c * Math.abs(d));
}

export function MessageList({
  messages,
  filtered,
  loading,
  error,
  searchQuery,
  onSearchChange,
  hasMore,
  loadingMore,
  onLoadMore,
  onDelete,
  onDeleteMany,
  onRefresh,
  total,
  modemNames,
  modems,
  deviceFilter,
  onDeviceFilterChange,
}: Props) {
  const [selectMode, setSelectMode] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(new Set());

  // 下拉刷新
  const [pull, setPull] = useState(0);
  const [refreshing, setRefreshing] = useState(false);
  const ptrStart = useRef<{ y: number; active: boolean }>({ y: 0, active: false });

  const toggleSelect = (msg: Message) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(msg.id)) next.delete(msg.id);
      else next.add(msg.id);
      return next;
    });
  };

  const exitSelect = () => {
    setSelectMode(false);
    setSelected(new Set());
  };

  const allSelected = filtered.length > 0 && selected.size === filtered.length;

  const onPtrDown = (e: React.PointerEvent) => {
    if (e.pointerType === "mouse" || refreshing) return;
    if (window.scrollY > 0) return;
    ptrStart.current = { y: e.clientY, active: true };
  };

  const onPtrMove = (e: React.PointerEvent) => {
    if (!ptrStart.current.active || refreshing) return;
    const dy = e.clientY - ptrStart.current.y;
    if (dy <= 0 || window.scrollY > 0) {
      setPull(0);
      return;
    }
    setPull(rubberband(dy));
  };

  const onPtrUp = async () => {
    if (!ptrStart.current.active) return;
    ptrStart.current.active = false;
    if (pull >= PTR_THRESHOLD * 0.7 && !refreshing) {
      setRefreshing(true);
      setPull(PTR_THRESHOLD * 0.6);
      try {
        await onRefresh();
      } finally {
        setRefreshing(false);
        setPull(0);
      }
    } else {
      setPull(0);
    }
  };

  const deviceOptions = modems.map((m) => ({
    imei: m.imei,
    label: modemNames[m.imei] || m.imei.slice(-4),
  }));

  return (
    <div
      className="w-full"
      onPointerDown={onPtrDown}
      onPointerMove={onPtrMove}
      onPointerUp={onPtrUp}
      onPointerCancel={onPtrUp}
      style={{ touchAction: "pan-y" }}
    >
      {/* 下拉刷新指示器 */}
      <div className="ptr-indicator" aria-hidden={pull === 0 && !refreshing}>
        <div
          style={{
            transform: `translateY(${pull / 2}px) scale(${Math.min(1, pull / PTR_THRESHOLD)})`,
            opacity: Math.min(1, pull / (PTR_THRESHOLD * 0.6)),
            transition: ptrStart.current.active ? "none" : "all 260ms var(--ease-out)",
          }}
        >
          <div
            className="spinner !h-5 !w-5"
            style={{
              animationPlayState: refreshing ? "running" : "paused",
              transform: refreshing ? undefined : `rotate(${pull * 2.4}deg)`,
            }}
          />
        </div>
      </div>

      <div
        style={{
          transform: pull > 0 ? `translateY(${pull}px)` : undefined,
          transition: ptrStart.current.active ? "none" : "transform 320ms var(--ease-drawer)",
        }}
      >
        <div className="mb-5 flex items-center justify-between gap-4">
          <p className="footnote">实时接收 · 本地存储</p>
          <div className="flex items-center gap-2">
            {!loading && !error && messages.length > 0 && (
              <button
                type="button"
                className="btn !px-3 !py-1 text-[0.8125rem]"
                onClick={() => (selectMode ? exitSelect() : setSelectMode(true))}
              >
                {selectMode ? "完成" : "选择"}
              </button>
            )}
            {!loading && !error && (
              <div className="rounded-full bg-[var(--fill-tertiary)] px-3 py-1 text-[0.8125rem] font-semibold tabular-nums text-[var(--label-secondary)]">
                {filtered.length}
                <span className="text-[var(--label-tertiary)]"> / {total}</span>
              </div>
            )}
          </div>
        </div>

        {!loading && !error && messages.length > 0 && (
          <div className="relative mb-3">
            <IconSearch
              size={18}
              className="pointer-events-none absolute top-1/2 left-3.5 -translate-y-1/2 text-[var(--label-tertiary)]"
            />
            <input
              type="search"
              className="field"
              style={{ paddingLeft: "2.5rem" }}
              placeholder="搜索内容、号码或设备"
              value={searchQuery}
              onChange={(e) => onSearchChange(e.target.value)}
            />
          </div>
        )}

        {/* 设备筛选芯片：多设备时展示 */}
        {!loading && !error && messages.length > 0 && deviceOptions.length > 1 && (
          <div className="filter-row mb-4">
            <button
              type="button"
              className="filter-chip"
              data-active={deviceFilter === ""}
              onClick={() => onDeviceFilterChange("")}
            >
              全部设备
            </button>
            {deviceOptions.map((d) => (
              <button
                key={d.imei}
                type="button"
                className="filter-chip"
                data-active={deviceFilter === d.imei}
                onClick={() =>
                  onDeviceFilterChange(deviceFilter === d.imei ? "" : d.imei)
                }
              >
                {d.label}
              </button>
            ))}
          </div>
        )}

        {loading && <MessageSkeleton />}

        {!loading && error && (
          <div
            className="flex items-start gap-3 rounded-[var(--radius-lg)] px-4 py-4"
            style={{
              background: "color-mix(in srgb, var(--red) 10%, transparent)",
              color: "var(--red)",
            }}
          >
            <IconAlert size={20} className="mt-0.5 shrink-0" />
            <p className="text-[0.9375rem] font-medium">{error}</p>
          </div>
        )}

        {!loading && !error && messages.length === 0 && (
          <div className="glass-card empty-state flex flex-col items-center justify-center gap-3 px-5 py-20 text-center">
            <div className="flex h-14 w-14 items-center justify-center rounded-full bg-[var(--fill-tertiary)] text-[var(--label-tertiary)]">
              <IconInboxLarge />
            </div>
            <div>
              <p className="headline">暂无短信</p>
              <p className="footnote mt-1">新消息会通过 WebSocket 实时出现在这里</p>
            </div>
          </div>
        )}

        {!loading && !error && messages.length > 0 && filtered.length === 0 && (
          <div className="glass-card empty-state flex flex-col items-center justify-center gap-3 px-5 py-16 text-center">
            <IconSearch size={28} className="text-[var(--label-tertiary)]" />
            <p className="headline">无匹配结果</p>
            <p className="footnote">试试其他关键词</p>
          </div>
        )}

        {!loading && !error && filtered.length > 0 && (
          <div className="stagger flex flex-col gap-3">
            {filtered.map((msg) => (
              <MessageCard
                key={msg.id}
                msg={msg}
                modemNames={modemNames}
                selectMode={selectMode}
                selected={selected.has(msg.id)}
                onToggleSelect={toggleSelect}
                onDelete={onDelete}
              />
            ))}

            {hasMore && !searchQuery.trim() && !selectMode && (
              <div className="pt-2 pb-6 text-center">
                <button
                  type="button"
                  className="btn"
                  disabled={loadingMore}
                  onClick={onLoadMore}
                >
                  {loadingMore ? "加载中…" : "加载更多"}
                </button>
              </div>
            )}
          </div>
        )}

        {/* 多选操作栏 */}
        {selectMode && (
          <div className="sticky-action">
            <div className="flex gap-2">
              <button
                type="button"
                className="btn flex-1"
                onClick={() => {
                  setSelected(
                    allSelected ? new Set() : new Set(filtered.map((m) => m.id)),
                  );
                }}
              >
                {allSelected ? "取消全选" : "全选"}
              </button>
              <button
                type="button"
                className="btn btn-danger flex-1"
                disabled={selected.size === 0}
                onClick={() => {
                  onDeleteMany([...selected]);
                  exitSelect();
                }}
              >
                删除所选{selected.size > 0 ? ` (${selected.size})` : ""}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
