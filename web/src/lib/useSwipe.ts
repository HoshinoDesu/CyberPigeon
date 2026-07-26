"use client";

import { useRef, useState } from "react";

const ACTION_W = 80; // 删除动作区宽度 px
const OPEN_THRESHOLD = 40;
const VELOCITY_COMMIT = 200; // px/s，速度符号优先于位置

function rubberband(over: number, dim = 120, c = 0.55): number {
  return (over * dim * c) / (dim + c * Math.abs(over));
}

/**
 * 消息卡左滑露出删除按钮。
 * 1:1 跟踪 + 越界 rubber-band + 松手按速度符号决定提交/回弹。
 */
export function useSwipe(onOpenChange?: (open: boolean) => void) {
  const [offset, setOffset] = useState(0);
  const [dragging, setDragging] = useState(false);
  const openRef = useRef(false);

  const start = useRef({ x: 0, y: 0, base: 0 });
  const history = useRef<{ x: number; t: number }[]>([]);
  const axis = useRef<"none" | "x" | "y">("none");

  const settle = (open: boolean) => {
    openRef.current = open;
    setOffset(open ? -ACTION_W : 0);
    onOpenChange?.(open);
  };

  const onPointerDown = (e: React.PointerEvent) => {
    if (e.pointerType === "mouse") return; // 桌面用 hover 删除
    start.current = { x: e.clientX, y: e.clientY, base: openRef.current ? -ACTION_W : 0 };
    history.current = [{ x: e.clientX, t: performance.now() }];
    axis.current = "none";
    setDragging(true);
  };

  const onPointerMove = (e: React.PointerEvent) => {
    if (!dragging) return;
    const dx = e.clientX - start.current.x;
    const dy = e.clientY - start.current.y;

    // ~10px 滞回确定方向，纵向滚动让位
    if (axis.current === "none") {
      if (Math.abs(dx) < 10 && Math.abs(dy) < 10) return;
      axis.current = Math.abs(dx) > Math.abs(dy) ? "x" : "y";
      if (axis.current === "x") {
        (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
      }
    }
    if (axis.current !== "x") return;

    history.current.push({ x: e.clientX, t: performance.now() });
    if (history.current.length > 5) history.current.shift();

    let next = start.current.base + dx;
    if (next > 0) next = rubberband(next); // 右越界阻尼
    if (next < -ACTION_W) next = -ACTION_W + rubberband(next + ACTION_W); // 左越界阻尼
    setOffset(next);
  };

  const onPointerUp = () => {
    if (!dragging) return;
    setDragging(false);
    if (axis.current !== "x") {
      setOffset(openRef.current ? -ACTION_W : 0);
      return;
    }

    // 末段速度（px/s）
    const h = history.current;
    let velocity = 0;
    if (h.length >= 2) {
      const a = h[0];
      const b = h[h.length - 1];
      const dt = b.t - a.t;
      if (dt > 0) velocity = ((b.x - a.x) / dt) * 1000;
    }

    // 速度符号优先，其次位置
    if (velocity < -VELOCITY_COMMIT) settle(true);
    else if (velocity > VELOCITY_COMMIT) settle(false);
    else settle(offset < -OPEN_THRESHOLD);
  };

  const close = () => settle(false);

  return {
    offset,
    dragging,
    close,
    handlers: {
      onPointerDown,
      onPointerMove,
      onPointerUp,
      onPointerCancel: onPointerUp,
    },
  };
}
