"use client";

import { useState } from "react";
import { formatRelativeTime } from "@/lib/format";
import { copyText, extractOtp } from "@/lib/otp";
import type { Message } from "@/lib/types";
import { useSwipe } from "@/lib/useSwipe";
import { IconCheck, IconCopy, IconTrash } from "./icons";

function OtpChip({ code }: { code: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      type="button"
      className="otp-chip"
      data-copied={copied}
      onClick={async () => {
        // 分组码（983-497）复制为纯数字，便于粘贴到验证码输入框
        const digits = code.replace(/[- ]/g, "");
        const ok = await copyText(/^\d+$/.test(digits) ? digits : code);
        if (ok) {
          setCopied(true);
          window.setTimeout(() => setCopied(false), 1600);
        }
      }}
      aria-label={`复制验证码 ${code}`}
    >
      <span className="mono text-[0.9375rem] font-bold tracking-[0.12em]">
        {code}
      </span>
      <span className="otp-chip-icon">
        {copied ? <IconCheck size={14} /> : <IconCopy size={14} />}
      </span>
      <span className="text-[0.6875rem] font-semibold">
        {copied ? "已复制" : "复制"}
      </span>
    </button>
  );
}

type Props = {
  msg: Message;
  modemNames: Record<string, string>;
  selectMode: boolean;
  selected: boolean;
  onToggleSelect: (msg: Message) => void;
  onDelete: (msg: Message) => void;
};

export function MessageCard({
  msg,
  modemNames,
  selectMode,
  selected,
  onToggleSelect,
  onDelete,
}: Props) {
  const swipe = useSwipe();
  const otp = extractOtp(msg.text);

  if (selectMode) {
    return (
      <button
        type="button"
        className="row-press w-full text-left"
        onClick={() => onToggleSelect(msg)}
        aria-pressed={selected}
      >
        <article className="msg-card glass-card flex items-start gap-3 p-4 sm:p-5">
          <span className="select-circle mt-0.5" data-checked={selected}>
            <IconCheck size={13} />
          </span>
          <div className="min-w-0 flex-1">
            <div className="mb-2 flex items-start justify-between gap-3">
              <div className="min-w-0">
                <div className="headline truncate">{msg.number || "未知号码"}</div>
                <div className="caption mono mt-1 truncate">
                  {msg.modem || "—"}
                  {msg.modem && modemNames[msg.modem] && (
                    <span className="text-[var(--label-tertiary)]">
                      {" "}
                      · {modemNames[msg.modem]}
                    </span>
                  )}
                </div>
              </div>
              <time className="caption whitespace-nowrap">
                {formatRelativeTime(msg.timestamp)}
              </time>
            </div>
            <div className="rounded-[var(--radius-md)] bg-[var(--fill-tertiary)] px-3.5 py-3 text-[0.9375rem] leading-relaxed break-words whitespace-pre-wrap text-[var(--label)]">
              {msg.text || "（空消息）"}
            </div>
          </div>
        </article>
      </button>
    );
  }

  const revealed = swipe.offset < -1 || swipe.dragging;

  return (
    <div className="swipe-item">
      <div
        className="swipe-action"
        style={{
          opacity: revealed ? 1 : 0,
          pointerEvents: revealed ? "auto" : "none",
          transition: "opacity 140ms var(--ease-out)",
        }}
      >
        <button
          type="button"
          className="pressable flex h-full w-full items-center justify-center"
          aria-label="删除"
          onClick={() => {
            swipe.close();
            onDelete(msg);
          }}
        >
          <IconTrash size={20} />
        </button>
      </div>
      <div
        className="swipe-content"
        style={{
          transform: `translateX(${swipe.offset}px)`,
          transition: swipe.dragging
            ? "none"
            : "transform 320ms var(--ease-drawer)",
        }}
        {...swipe.handlers}
      >
        <article className="msg-card glass-card p-4 sm:p-5">
          <div className="mb-3 flex items-start justify-between gap-3">
            <div className="min-w-0">
              <div className="headline truncate">{msg.number || "未知号码"}</div>
              <div className="caption mono mt-1 truncate">
                {msg.modem || "—"}
                {msg.modem && modemNames[msg.modem] && (
                  <span className="text-[var(--label-tertiary)]">
                    {" "}
                    · {modemNames[msg.modem]}
                  </span>
                )}
              </div>
            </div>
            <div className="flex shrink-0 items-center gap-1.5">
              <time className="caption whitespace-nowrap">
                {formatRelativeTime(msg.timestamp)}
              </time>
              <button
                type="button"
                className="btn btn-icon btn-ghost msg-delete"
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
          {otp && (
            <div className="mt-2.5">
              <OtpChip code={otp} />
            </div>
          )}
        </article>
      </div>
    </div>
  );
}
