"use client";

import { getSignalColor } from "@/lib/format";

/** iOS 风格离散信号格：4 格阶梯柱状 */
export function SignalBars({ quality }: { quality: number }) {
  const active =
    quality >= 80 ? 4 : quality >= 60 ? 3 : quality >= 40 ? 2 : quality > 0 ? 1 : 0;
  const color = getSignalColor(quality);
  return (
    <span className="signal-bars" role="img" aria-label={`信号 ${quality}%`}>
      {[0, 1, 2, 3].map((i) => (
        <span
          key={i}
          className="signal-bar"
          style={{
            height: `${40 + i * 20}%`,
            background: i < active ? color : "var(--signal-none)",
          }}
        />
      ))}
    </span>
  );
}
