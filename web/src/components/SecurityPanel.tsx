"use client";

import { PasswordInput } from "./PasswordInput";

type Props = {
  currentPassword: string;
  newPassword: string;
  confirmPassword: string;
  error: string;
  changing: boolean;
  onChange: (
    field: "current_password" | "new_password" | "confirm_password",
    value: string,
  ) => void;
  onSubmit: () => void;
};

export function SecurityPanel({
  currentPassword,
  newPassword,
  confirmPassword,
  error,
  changing,
  onChange,
  onSubmit,
}: Props) {
  return (
    <div className="rounded-[var(--radius-lg)] bg-[var(--fill-tertiary)] p-4">
      <div className="mb-4">
        <h3 className="headline">管理密码</h3>
        <p className="footnote mt-1">
          修改后其他会话会全部失效，当前浏览器会自动续签。
        </p>
      </div>

      <div className="flex flex-col gap-3.5">
        <div>
          <label className="label">当前密码</label>
          <PasswordInput
            value={currentPassword}
            onChange={(v) => onChange("current_password", v)}
            placeholder="输入当前密码"
            autoComplete="current-password"
          />
        </div>
        <div>
          <label className="label">新密码</label>
          <PasswordInput
            value={newPassword}
            onChange={(v) => onChange("new_password", v)}
            placeholder="至少 6 位"
            autoComplete="new-password"
          />
        </div>
        <div>
          <label className="label">确认新密码</label>
          <PasswordInput
            value={confirmPassword}
            onChange={(v) => onChange("confirm_password", v)}
            placeholder="再次输入新密码"
            autoComplete="new-password"
          />
        </div>

        {error && (
          <div
            className="rounded-[var(--radius-sm)] px-3 py-2.5 text-[0.875rem] font-medium"
            style={{
              background: "color-mix(in srgb, var(--red) 12%, transparent)",
              color: "var(--red)",
            }}
          >
            {error}
          </div>
        )}

        <button
          type="button"
          className="btn btn-primary mt-1 w-full py-3"
          disabled={changing}
          onClick={onSubmit}
        >
          {changing ? "修改中…" : "更新密码"}
        </button>
      </div>
    </div>
  );
}
