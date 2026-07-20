"use client";

import { getSignalClass, getSignalColor } from "@/lib/format";
import type { ModemInfo, Settings, UssdState } from "@/lib/types";
import { IconDevice, IconSend } from "./icons";

type Props = {
  modems: ModemInfo[];
  loading: boolean;
  error: string | null;
  ussdStates: Record<string, UssdState>;
  onUssdCodeChange: (imei: string, code: string) => void;
  onRunUssd: (modem: ModemInfo) => void;
  settings: Settings;
  settingsSaving: boolean;
  onSettingsChange: (next: Settings) => void;
  onSaveSettings: () => void;
};

function StateBox({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 rounded-[var(--radius-lg)] bg-[var(--fill-tertiary)] px-4 py-14 text-center">
      {children}
    </div>
  );
}

function getModemDisplayName(settings: Settings, imei: string): string {
  const entry = settings.modems.find((m) => m.imei === imei);
  return entry?.name || "";
}

function setModemDisplayName(settings: Settings, imei: string, name: string): Settings {
  const existing = settings.modems.filter((m) => m.imei !== imei);
  const next = name.trim()
    ? [...existing, { imei, name }]
    : existing;
  return { ...settings, modems: next };
}

export function DevicesPanel({
  modems,
  loading,
  error,
  ussdStates,
  onUssdCodeChange,
  onRunUssd,
  settings,
  settingsSaving,
  onSettingsChange,
  onSaveSettings,
}: Props) {
  if (loading) {
    return (
      <StateBox>
        <div className="spinner" />
        <p className="footnote">加载设备</p>
      </StateBox>
    );
  }

  if (error) {
    return (
      <div
        className="rounded-[var(--radius-md)] px-3 py-3 text-[0.875rem] font-medium"
        style={{
          background: "color-mix(in srgb, var(--red) 12%, transparent)",
          color: "var(--red)",
        }}
      >
        {error}
      </div>
    );
  }

  if (modems.length === 0) {
    return (
      <StateBox>
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-[var(--fill-secondary)] text-[var(--label-tertiary)]">
          <IconDevice size={22} />
        </div>
        <p className="headline">未检测到设备</p>
        <p className="footnote">请确认 ModemManager 与调制解调器状态</p>
      </StateBox>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      {/* Always On 全局开关 */}
      <section className="rounded-[var(--radius-lg)] bg-[var(--fill-tertiary)] p-4">
        <div className="flex items-center justify-between">
          <div>
            <div className="subhead">Always On</div>
            <p className="caption mt-0.5">自动启用处于 Disabled 状态的模块</p>
          </div>
          <label className="switch">
            <input
              type="checkbox"
              checked={settings.always_on_modems}
              onChange={(e) =>
                onSettingsChange({ ...settings, always_on_modems: e.target.checked })
              }
            />
            <span className="switch-track">
              <span className="switch-thumb" />
            </span>
          </label>
        </div>
      </section>

      {/* 设备名显示选项 */}
      <section className="rounded-[var(--radius-lg)] bg-[var(--fill-tertiary)] p-4">
        <div className="subhead mb-3">推送模板</div>
        <div className="flex flex-col gap-2">
          <label className="flex items-center gap-2 text-[0.875rem] text-[var(--label-secondary)]">
            <input
              type="checkbox"
              checked={settings.device_name_in_title}
              onChange={(e) =>
                onSettingsChange({
                  ...settings,
                  device_name_in_title: e.target.checked,
                })
              }
            />
            设备名显示在推送标题
          </label>
          <label className="flex items-center gap-2 text-[0.875rem] text-[var(--label-secondary)]">
            <input
              type="checkbox"
              checked={settings.device_name_in_body}
              onChange={(e) =>
                onSettingsChange({
                  ...settings,
                  device_name_in_body: e.target.checked,
                })
              }
            />
            设备名显示在推送正文
          </label>
        </div>
      </section>

      {modems.map((modem) => {
        const ussd = ussdStates[modem.imei] || {
          code: "",
          result: "",
          loading: false,
          error: "",
        };
        const displayName = getModemDisplayName(settings, modem.imei);
        return (
          <section
            key={modem.imei}
            className="rounded-[var(--radius-lg)] bg-[var(--fill-tertiary)] p-4"
          >
            <div className="mb-4 flex items-start justify-between gap-3">
              <div>
                <h3 className="headline">
                  {modem.model || modem.manufacturer || "未知设备"}
                </h3>
                <p className="caption mono mt-1">{modem.manufacturer || "—"}</p>
              </div>
              <div className="flex flex-col items-end gap-1">
                <span
                  className="mono text-[0.8125rem] font-semibold tabular-nums"
                  style={{ color: getSignalColor(modem.signal_quality) }}
                >
                  {modem.signal_quality}%
                </span>
                {modem.state && modem.state !== "registered" && modem.state !== "connected" && (
                  <span className="caption mono">{modem.state}</span>
                )}
              </div>
            </div>

            <div className="mb-4">
              <div className="signal-track mb-1.5">
                <div
                  className={`signal-fill ${getSignalClass(modem.signal_quality)}`}
                  style={{
                    width: `${Math.max(0, Math.min(100, modem.signal_quality))}%`,
                  }}
                />
              </div>
              <p className="caption">信号质量</p>
            </div>

            <div className="mb-4">
              {(
                [
                  ["IMEI", modem.imei],
                  ["ICCID", modem.iccid],
                  ["号码", modem.number],
                  ["运营商", modem.operator_name],
                ] as const
              ).map(([label, value]) => (
                <div key={label} className="list-row">
                  <span className="footnote">{label}</span>
                  <span className="meta-value max-w-[60%] text-right break-all">
                    {value || "—"}
                  </span>
                </div>
              ))}
            </div>

            <div className="mb-4 rounded-[var(--radius-md)] bg-[var(--bg-elevated)] p-3">
              <div className="subhead mb-2">模块名称</div>
              <input
                className="field"
                value={displayName}
                placeholder={modem.model || "未命名"}
                onChange={(e) =>
                  onSettingsChange(setModemDisplayName(settings, modem.imei, e.target.value))
                }
              />
              <p className="caption mt-1.5">推送消息中显示的模块名称，为空时使用全局设备名</p>
            </div>

            <div className="mb-4 rounded-[var(--radius-md)] bg-[var(--bg-elevated)] p-3">
              <div className="subhead mb-2">USSD</div>
              <div className="flex gap-2">
                <input
                  className="field"
                  value={ussd.code}
                  placeholder="*100#"
                  disabled={ussd.loading}
                  onChange={(e) => onUssdCodeChange(modem.imei, e.target.value)}
                />
                <button
                  type="button"
                  className="btn btn-primary shrink-0 px-3"
                  disabled={ussd.loading || !!ussd.error}
                  onClick={() => onRunUssd(modem)}
                  aria-label="发送 USSD"
                >
                  {ussd.loading ? (
                    <span className="spinner !h-4 !w-4 !border-[1.5px]" />
                  ) : (
                    <IconSend size={16} />
                  )}
                </button>
              </div>
              {ussd.error && (
                <p className="mt-2 text-[0.75rem] font-medium" style={{ color: "var(--red)" }}>
                  {ussd.error}
                </p>
              )}
              {ussd.result && (
                <pre className="mono mt-2 overflow-x-auto rounded-[var(--radius-sm)] bg-[var(--fill-tertiary)] p-2.5 text-[0.75rem] leading-relaxed whitespace-pre-wrap text-[var(--label-secondary)]">
                  {ussd.result}
                </pre>
              )}
            </div>
          </section>
        );
      })}

      <div className="sticky-action">
        <button
          type="button"
          className="btn btn-primary w-full"
          disabled={settingsSaving}
          onClick={onSaveSettings}
        >
          {settingsSaving ? "保存中…" : "保存设置"}
        </button>
      </div>
    </div>
  );
}
