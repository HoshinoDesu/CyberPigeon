"use client";

import { useEffect, useState } from "react";

type Props = {
  open: boolean;
  title: string;
  message: React.ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
};

export function ConfirmModal({
  open,
  title,
  message,
  confirmLabel = "确认",
  cancelLabel = "取消",
  danger,
  onConfirm,
  onCancel,
}: Props) {
  // 退出沿进入的同一路径反向播放，动画结束后再卸载
  const [render, setRender] = useState(open);
  const [closing, setClosing] = useState(false);

  useEffect(() => {
    if (open) {
      setRender(true);
      setClosing(false);
      return;
    }
    if (!render) return;
    setClosing(true);
    const timer = window.setTimeout(() => {
      setRender(false);
      setClosing(false);
    }, 200);
    return () => window.clearTimeout(timer);
  }, [open, render]);

  if (!render) return null;

  return (
    <div className="modal-scrim" data-closing={closing} onClick={onCancel}>
      <div
        className="modal"
        data-closing={closing}
        role="dialog"
        aria-modal="true"
        aria-labelledby="confirm-title"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 id="confirm-title" className="title mb-2">
          {title}
        </h2>
        <div className="mb-5 text-[0.9375rem] leading-relaxed text-[var(--label-secondary)]">
          {message}
        </div>
        <div className="grid grid-cols-2 gap-2">
          <button type="button" className="btn" onClick={onCancel}>
            {cancelLabel}
          </button>
          <button
            type="button"
            className={`btn ${danger ? "btn-danger" : "btn-primary"}`}
            onClick={onConfirm}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
