"use client";

import type { ToastItem } from "@/lib/types";

export function ToastStack({ toasts }: { toasts: ToastItem[] }) {
  return (
    <div className="toast-stack">
      {toasts.map((toast) => (
        <div
          key={toast.id}
          className="toast"
          data-type={toast.type}
          data-leaving={toast.leaving}
          role="status"
        >
          {toast.message}
        </div>
      ))}
    </div>
  );
}
