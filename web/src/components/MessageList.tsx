"use client";

import { formatRelativeTime } from "@/lib/format";
import type { Message } from "@/lib/types";
import { IconAlert, IconInboxLarge, IconSearch, IconTrash } from "./icons";

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
  total: number;
  modemNames: Record<string, string>;
};

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
  total,
  modemNames,
}: Props) {
  return (
    <div className="mx-auto w-full max-w-[44rem]">
      <div className="mb-5 flex items-center justify-between gap-4">
        <p className="footnote">实时接收 · 本地存储</p>
        {!loading && !error && (
          <div className="rounded-full bg-[var(--fill-tertiary)] px-3 py-1 text-[0.8125rem] font-semibold tabular-nums text-[var(--label-secondary)]">
            {filtered.length}
            <span className="text-[var(--label-tertiary)]"> / {total}</span>
          </div>
        )}
      </div>

      {!loading && !error && messages.length > 0 && (
        <div className="relative mb-5">
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

      {loading && (
        <div className="glass-card flex flex-col items-center justify-center gap-3 px-5 py-20">
          <div className="spinner" />
          <p className="footnote">正在加载消息</p>
        </div>
      )}

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
        <div className="glass-card flex flex-col items-center justify-center gap-3 px-5 py-20 text-center">
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
        <div className="glass-card flex flex-col items-center justify-center gap-3 px-5 py-16 text-center">
          <IconSearch size={28} className="text-[var(--label-tertiary)]" />
          <p className="headline">无匹配结果</p>
          <p className="footnote">试试其他关键词</p>
        </div>
      )}

      {!loading && !error && filtered.length > 0 && (
        <div className="stagger flex flex-col gap-3">
          {filtered.map((msg) => (
            <article key={msg.id} className="msg-card glass-card p-4 sm:p-5">
              <div className="mb-3 flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="headline truncate">{msg.number || "未知号码"}</div>
                  <div className="caption mono mt-1 truncate">
                    {msg.modem || "—"}
                    {msg.modem && modemNames[msg.modem] && (
                      <span className="text-[var(--label-tertiary)]"> · {modemNames[msg.modem]}</span>
                    )}
                  </div>
                </div>
                <div className="flex shrink-0 items-center gap-1.5">
                  <time className="caption whitespace-nowrap">
                    {formatRelativeTime(msg.timestamp)}
                  </time>
                  <button
                    type="button"
                    className="btn btn-icon btn-ghost"
                    aria-label="删除"
                    onClick={() => onDelete(msg)}
                  >
                    <IconTrash size={17} />
                  </button>
                </div>
              </div>
              <div className="rounded-[var(--radius-md)] bg-[var(--fill-tertiary)] px-3.5 py-3 text-[0.9375rem] leading-relaxed break-words whitespace-pre-wrap text-[var(--label)]">
                {msg.text || "（空消息）"}
              </div>
            </article>
          ))}

          {hasMore && !searchQuery.trim() && (
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
    </div>
  );
}
