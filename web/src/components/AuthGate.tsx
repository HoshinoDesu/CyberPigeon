"use client";

import { PasswordInput } from "./PasswordInput";

type Props = {
  requiresSetup: boolean;
  message?: string;
  error?: string;
  password: string;
  passwordConfirm: string;
  submitting: boolean;
  onPasswordChange: (v: string) => void;
  onPasswordConfirmChange: (v: string) => void;
  onSubmit: () => void;
};

export function AuthGate({
  requiresSetup,
  message,
  error,
  password,
  passwordConfirm,
  submitting,
  onPasswordChange,
  onPasswordConfirmChange,
  onSubmit,
}: Props) {
  const submit = () => {
    if (!submitting) onSubmit();
  };

  return (
    <div className="relative flex min-h-dvh items-center justify-center px-5 py-10">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0"
        style={{
          background:
            "radial-gradient(ellipse 70% 45% at 50% 0%, color-mix(in srgb, var(--label) 7%, transparent), transparent 70%)",
        }}
      />

      <div className="relative w-full max-w-[22rem]">
        <div className="mb-7 text-center">
          <p className="caption mb-2 tracking-[0.18em] uppercase">Console</p>
          <h1 className="display mb-2">CyberPigeon</h1>
          <p className="footnote mx-auto max-w-[18rem]">
            {requiresSetup
              ? "首次使用请设置管理密码，用于保护控制台与 API。"
              : message || "输入管理密码以继续。"}
          </p>
        </div>

        <div className="glass-card p-5 shadow-[var(--shadow-float)]">
          <div className="mb-4">
            <label className="label">密码</label>
            <PasswordInput
              value={password}
              onChange={onPasswordChange}
              placeholder="至少 6 位"
              autoComplete={requiresSetup ? "new-password" : "current-password"}
              onKeyDown={(e) => {
                if (e.key === "Enter") submit();
              }}
            />
          </div>

          {requiresSetup && (
            <div className="mb-4">
              <label className="label">确认密码</label>
              <PasswordInput
                value={passwordConfirm}
                onChange={onPasswordConfirmChange}
                placeholder="再次输入"
                autoComplete="new-password"
                onKeyDown={(e) => {
                  if (e.key === "Enter") submit();
                }}
              />
            </div>
          )}

          {error && (
            <div
              className="mb-4 rounded-[var(--radius-sm)] px-3 py-2.5 text-[0.875rem] font-medium"
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
            className="btn btn-primary w-full py-3"
            disabled={submitting}
            onClick={submit}
          >
            {submitting
              ? requiresSetup
                ? "设置中…"
                : "登录中…"
              : requiresSetup
                ? "设置并进入"
                : "登录"}
          </button>
        </div>
      </div>
    </div>
  );
}
