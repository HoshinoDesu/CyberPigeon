"use client";

import { getSignalColor } from "@/lib/format";
import type { ModemInfo, Settings, UssdState } from "@/lib/types";
import { EMPTY_USSD_STATE } from "@/lib/types";
import { IconDevice, IconSend } from "./icons";
import { SignalBars } from "./SignalBars";

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

const MODEM_STATE_LABELS: Record<string, string> = {
  failed: "故障",
  unknown: "未知",
  initializing: "初始化中",
  locked: "已锁定",
  disabled: "已禁用",
  disabling: "禁用中",
  enabling: "启用中",
  enabled: "已启用",
  searching: "搜索网络",
  registered: "已注册",
  disconnecting: "断开中",
  connecting: "连接中",
  connected: "已连接",
};

/** 需要用户留意的异常状态，用红色 badge 强调 */
const MODEM_STATE_ALERT = new Set(["failed", "locked", "disabled"]);

function StateBox({ children }: { children: React.ReactNode }) {
  return (
    <div className="empty-state flex flex-col items-center justify-center gap-3 rounded-[var(--radius-lg)] bg-[var(--fill-tertiary)] px-4 py-14 text-center">
      {children}
    </div>
  );
}

function Switch({
  checked,
  onChange,
  label,
}: {
  checked: boolean;
  onChange: (v: boolean) => void;
  label: string;
}) {
  return (
    <label className="switch" title={label} aria-label={label}>
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
      />
      <span className="switch-track">
        <span className="switch-thumb" />
      </span>
    </label>
  );
}

function getModemNote(settings: Settings, imei: string): string {
  return settings.modems.find((m) => m.imei === imei)?.name || "";
}

function setModemNote(settings: Settings, imei: string, name: string): Settings {
  const exists = settings.modems.some((m) => m.imei === imei);
  const next = exists
    ? settings.modems.map((m) => (m.imei === imei ? { ...m, name } : m))
    : [...settings.modems, { imei, name }];
  return { ...settings, modems: next.filter((m) => m.name.trim() !== "") };
}

function ModemStateBadge({ state }: { state: string }) {
  if (!state || state === "registered" || state === "connected") return null;
  const label = MODEM_STATE_LABELS[state] || state;
  const alert = MODEM_STATE_ALERT.has(state);
  return (
    <span
      className="rounded-full px-2 py-0.5 text-[0.6875rem] font-semibold whitespace-nowrap"
      style={
        alert
          ? {
              background: "color-mix(in srgb, var(--red) 14%, transparent)",
              color: "var(--red)",
            }
          : {
              background: "var(--fill-secondary)",
              color: "var(--label-secondary)",
            }
      }
    >
      {label}
    </span>
  );
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
      <div className="flex flex-col gap-4">
        {[0, 1].map((i) => (
          <div
            key={i}
            className="rounded-[var(--radius-lg)] bg-[var(--fill-tertiary)] p-4"
            style={{ opacity: 1 - i * 0.35 }}
          >
            <div className="mb-4 flex items-start justify-between gap-3">
              <div className="flex-1">
                <div className="skeleton h-5 w-48" />
                <div className="skeleton mt-2 h-3.5 w-36" />
              </div>
              <div className="skeleton h-4 w-16" />
            </div>
            <div className="flex flex-col gap-2.5">
              <div className="skeleton h-3.5 w-full" />
              <div className="skeleton h-3.5 w-full" />
              <div className="skeleton h-3.5 w-2/3" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className="stagger flex flex-col gap-4">
      {/* 全局设置：无论有无设备均可配置 */}
      <section className="rounded-[var(--radius-lg)] bg-[var(--fill-tertiary)] p-4">
        <div className="flex items-center justify-between gap-3">
          <div className="min-w-0">
            <div className="subhead">Always On</div>
            <p className="caption mt-0.5">自动启用处于已禁用状态的模块</p>
          </div>
          <Switch
            checked={settings.always_on_modems}
            onChange={(v) => onSettingsChange({ ...settings, always_on_modems: v })}
            label="Always On"
          />
        </div>

        <div className="mt-4 border-t border-[var(--hairline)] pt-4">
          <div className="subhead mb-3">推送模板</div>
          <div className="flex flex-col gap-3">
            <div className="flex items-center justify-between gap-3">
              <span className="text-[0.875rem] text-[var(--label-secondary)]">
                设备名显示在推送标题
              </span>
              <Switch
                checked={settings.device_name_in_title}
                onChange={(v) =>
                  onSettingsChange({ ...settings, device_name_in_title: v })
                }
                label="设备名显示在推送标题"
              />
            </div>
            <div className="flex items-center justify-between gap-3">
              <span className="text-[0.875rem] text-[var(--label-secondary)]">
                设备名显示在推送正文
              </span>
              <Switch
                checked={settings.device_name_in_body}
                onChange={(v) =>
                  onSettingsChange({ ...settings, device_name_in_body: v })
                }
                label="设备名显示在推送正文"
              />
            </div>
          </div>
        </div>
      </section>

      {error && (
        <div
          className="rounded-[var(--radius-md)] px-3 py-3 text-[0.875rem] font-medium"
          style={{
            background: "color-mix(in srgb, var(--red) 12%, transparent)",
            color: "var(--red)",
          }}
        >
          {error}
        </div>
      )}

      {!error && modems.length === 0 && (
        <StateBox>
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-[var(--fill-secondary)] text-[var(--label-tertiary)]">
            <IconDevice size={22} />
          </div>
          <p className="headline">未检测到设备</p>
          <p className="footnote">请确认 ModemManager 与调制解调器状态</p>
        </StateBox>
      )}

      {modems.map((modem) => {
        const ussd = ussdStates[modem.imei] || EMPTY_USSD_STATE;
        const note = getModemNote(settings, modem.imei);
        const hardwareName = modem.model || modem.manufacturer || "未知设备";
        return (
          <section
            key={modem.imei}
            className="rounded-[var(--radius-lg)] bg-[var(--fill-tertiary)] p-4"
          >
            <div className="mb-4 flex items-start justify-between gap-3">
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <h3 className="headline truncate-safe">
                    {note || hardwareName}
                  </h3>
                  <ModemStateBadge state={modem.state} />
                </div>
                <p className="caption mono mt-1 truncate-safe">
                  {note ? hardwareName : modem.manufacturer || "—"}
                </p>
              </div>
              <span className="flex shrink-0 items-center gap-1.5">
                <SignalBars quality={modem.signal_quality} />
                <span
                  className="mono text-[0.8125rem] font-semibold tabular-nums"
                  style={{ color: getSignalColor(modem.signal_quality) }}
                >
                  {modem.signal_quality}%
                </span>
              </span>
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
              <div className="subhead mb-2">模块备注名</div>
              <input
                className="field"
                value={note}
                placeholder={hardwareName}
                onChange={(e) =>
                  onSettingsChange(setModemNote(settings, modem.imei, e.target.value))
                }
              />
              <p className="caption mt-1.5">
                推送消息中显示的模块名称，为空时使用全局设备名
              </p>
            </div>

            <div className="rounded-[var(--radius-md)] bg-[var(--bg-elevated)] p-3">
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
