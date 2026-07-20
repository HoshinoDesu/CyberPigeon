"use client";

import { useEffect, useRef, useState } from "react";
import {
  CHANNEL_OPTIONS,
  fromStr,
  getChannelLabel,
  toStr,
} from "@/lib/channels";
import type { ChannelConfig } from "@/lib/types";
import { IconChevron, IconPlus, IconTrash } from "./icons";
import { PasswordInput } from "./PasswordInput";

type Props = {
  channels: ChannelConfig[];
  loading: boolean;
  error: string | null;
  saving: boolean;
  testing: boolean;
  newChannelType: string;
  onNewChannelTypeChange: (v: string) => void;
  onChange: (index: number, next: ChannelConfig) => void;
  onAdd: () => void;
  onRemove: (index: number) => void;
  onSave: () => void;
  onTest: () => void;
};

function channelSummary(ch: ChannelConfig): string {
  switch (ch.type) {
    case "email":
      return ch.host || ch.from || "未配置 SMTP";
    case "bark":
      return ch.endpoint || "未配置推送地址";
    case "gotify":
      return ch.endpoint || "未配置服务器";
    case "serverchan":
      return ch.send_key ? "已配置 SendKey" : "未配置 SendKey";
    case "webhook":
      return ch.url || "未配置 URL";
    case "wecom":
      return ch.corp_id || "未配置企业微信";
    case "feishu":
      return ch.app_id || ch.receive_id || "未配置飞书";
    case "dingtalk":
      return ch.webhook_url || "未配置 Webhook";
    case "telegram":
      return ch.chat_id || (ch.bot_token ? "已配置 Bot" : "未配置 Telegram");
    default:
      return ch.type;
  }
}

function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <label className="label">{label}</label>
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
  onChange: (next: boolean) => void;
  label?: string;
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

function ChannelFields({
  ch,
  onChange,
}: {
  ch: ChannelConfig;
  onChange: (next: ChannelConfig) => void;
}) {
  switch (ch.type) {
    case "email":
      return (
        <>
          <Field label="SMTP 服务器">
            <input
              className="field"
              value={ch.host || ""}
              placeholder="smtp.example.com"
              onChange={(e) => onChange({ ...ch, host: e.target.value })}
            />
          </Field>
          <Field label="端口">
            <input
              type="number"
              className="field"
              value={ch.port ?? ""}
              placeholder="587"
              onChange={(e) =>
                onChange({ ...ch, port: Number(e.target.value) || 0 })
              }
            />
          </Field>
          <label className="flex items-center gap-2 text-[0.875rem] text-[var(--label-secondary)]">
            <input
              type="checkbox"
              checked={!!ch.use_tls}
              onChange={(e) => onChange({ ...ch, use_tls: e.target.checked })}
            />
            使用 TLS
          </label>
          <Field label="用户名">
            <input
              className="field"
              value={ch.username || ""}
              onChange={(e) => onChange({ ...ch, username: e.target.value })}
            />
          </Field>
          <Field label="密码">
            <PasswordInput
              value={ch.password || ""}
              placeholder="••••••"
              onChange={(v) => onChange({ ...ch, password: v })}
            />
          </Field>
          <Field label="发件人">
            <input
              className="field"
              value={ch.from || ""}
              onChange={(e) => onChange({ ...ch, from: e.target.value })}
            />
          </Field>
          <Field label="收件人（逗号分隔）">
            <input
              className="field"
              value={toStr(ch.to)}
              onChange={(e) => onChange({ ...ch, to: fromStr(e.target.value) })}
            />
          </Field>
        </>
      );
    case "bark":
      return (
        <>
          <Field label="推送地址">
            <input
              className="field"
              value={ch.endpoint || ""}
              placeholder="https://api.day.app/your_key"
              onChange={(e) => onChange({ ...ch, endpoint: e.target.value })}
            />
          </Field>
          <Field label="标题（可选）">
            <input
              className="field"
              value={ch.title || ""}
              onChange={(e) => onChange({ ...ch, title: e.target.value })}
            />
          </Field>
        </>
      );
    case "gotify":
      return (
        <>
          <Field label="服务器地址">
            <input
              className="field"
              value={ch.endpoint || ""}
              onChange={(e) => onChange({ ...ch, endpoint: e.target.value })}
            />
          </Field>
          <Field label="Token">
            <PasswordInput
              value={ch.token || ""}
              onChange={(v) => onChange({ ...ch, token: v })}
            />
          </Field>
          <Field label="优先级 (0-10)">
            <input
              type="number"
              min={0}
              max={10}
              className="field"
              value={ch.priority ?? ""}
              onChange={(e) =>
                onChange({ ...ch, priority: Number(e.target.value) || 0 })
              }
            />
          </Field>
        </>
      );
    case "serverchan":
      return (
        <Field label="SendKey">
          <PasswordInput
            value={ch.send_key || ""}
            onChange={(v) => onChange({ ...ch, send_key: v })}
          />
        </Field>
      );
    case "webhook":
      return (
        <>
          <Field label="URL">
            <input
              className="field"
              value={ch.url || ""}
              onChange={(e) => onChange({ ...ch, url: e.target.value })}
            />
          </Field>
          <Field label="请求方法">
            <select
              className="field"
              value={ch.method || "POST"}
              onChange={(e) => onChange({ ...ch, method: e.target.value })}
            >
              <option value="POST">POST</option>
              <option value="GET">GET</option>
            </select>
          </Field>
          <Field label="请求头 (JSON)">
            <textarea
              className="field"
              rows={3}
              value={ch.headers_json || ""}
              placeholder='{"Authorization":"Bearer …"}'
              onChange={(e) =>
                onChange({ ...ch, headers_json: e.target.value })
              }
            />
          </Field>
          <label className="flex items-center gap-2 text-[0.875rem] text-[var(--label-secondary)]">
            <input
              type="checkbox"
              checked={!!ch.allow_private_network}
              onChange={(e) =>
                onChange({ ...ch, allow_private_network: e.target.checked })
              }
            />
            允许访问私网地址
          </label>
        </>
      );
    case "wecom":
      return (
        <>
          <Field label="Corp ID">
            <input
              className="field"
              value={ch.corp_id || ""}
              onChange={(e) => onChange({ ...ch, corp_id: e.target.value })}
            />
          </Field>
          <Field label="Corp Secret">
            <PasswordInput
              value={ch.corp_secret || ""}
              onChange={(v) => onChange({ ...ch, corp_secret: v })}
            />
          </Field>
          <Field label="Agent ID">
            <input
              type="number"
              className="field"
              value={ch.agent_id ?? ""}
              onChange={(e) =>
                onChange({ ...ch, agent_id: Number(e.target.value) || 0 })
              }
            />
          </Field>
          <Field label="接收人（可选）">
            <input
              className="field"
              value={ch.to_user || ""}
              onChange={(e) => onChange({ ...ch, to_user: e.target.value })}
            />
          </Field>
        </>
      );
    case "feishu":
      return (
        <>
          <Field label="App ID">
            <input
              className="field"
              value={ch.app_id || ""}
              onChange={(e) => onChange({ ...ch, app_id: e.target.value })}
            />
          </Field>
          <Field label="App Secret">
            <PasswordInput
              value={ch.app_secret || ""}
              onChange={(v) => onChange({ ...ch, app_secret: v })}
            />
          </Field>
          <Field label="接收者 ID">
            <input
              className="field"
              value={ch.receive_id || ""}
              onChange={(e) => onChange({ ...ch, receive_id: e.target.value })}
            />
          </Field>
          <Field label="接收者 ID 类型">
            <select
              className="field"
              value={ch.receive_id_type || "open_id"}
              onChange={(e) =>
                onChange({ ...ch, receive_id_type: e.target.value })
              }
            >
              <option value="open_id">open_id</option>
              <option value="user_id">user_id</option>
              <option value="union_id">union_id</option>
              <option value="email">email</option>
              <option value="chat_id">chat_id</option>
            </select>
          </Field>
          <Field label="标题（可选）">
            <input
              className="field"
              value={ch.title || ""}
              onChange={(e) => onChange({ ...ch, title: e.target.value })}
            />
          </Field>
        </>
      );
    case "dingtalk":
      return (
        <>
          <Field label="Webhook URL">
            <input
              className="field"
              value={ch.webhook_url || ""}
              onChange={(e) => onChange({ ...ch, webhook_url: e.target.value })}
            />
          </Field>
          <Field label="签名密钥（可选）">
            <PasswordInput
              value={ch.sign_secret || ""}
              onChange={(v) => onChange({ ...ch, sign_secret: v })}
            />
          </Field>
          <Field label="标题（可选）">
            <input
              className="field"
              value={ch.title || ""}
              onChange={(e) => onChange({ ...ch, title: e.target.value })}
            />
          </Field>
        </>
      );
    case "telegram":
      return (
        <>
          <Field label="Bot Token">
            <PasswordInput
              value={ch.bot_token || ""}
              onChange={(v) => onChange({ ...ch, bot_token: v })}
            />
          </Field>
          <Field label="Chat ID">
            <input
              className="field"
              value={ch.chat_id || ""}
              onChange={(e) => onChange({ ...ch, chat_id: e.target.value })}
            />
          </Field>
          <Field label="API URL（可选）">
            <input
              className="field"
              value={ch.api_url || ""}
              placeholder="https://api.telegram.org"
              onChange={(e) => onChange({ ...ch, api_url: e.target.value })}
            />
          </Field>
        </>
      );
    default:
      return (
        <p className="footnote">未知通道类型：{ch.type}</p>
      );
  }
}

export function ChannelsPanel({
  channels,
  loading,
  error,
  saving,
  testing,
  newChannelType,
  onNewChannelTypeChange,
  onChange,
  onAdd,
  onRemove,
  onSave,
  onTest,
}: Props) {
  // Default collapsed. Only auto-expand when the user adds exactly one channel.
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  const prevLenRef = useRef(0);

  useEffect(() => {
    const prevLen = prevLenRef.current;
    const nextLen = channels.length;

    // Bulk load (0 -> N) or replace must stay collapsed.
    if (prevLen > 0 && nextLen === prevLen + 1 && nextLen > 0) {
      const index = nextLen - 1;
      const key = `${channels[index].type}-${index}`;
      setExpanded((prev) => ({ ...prev, [key]: true }));
    }

    if (nextLen < prevLen) {
      setExpanded((prev) => {
        const next: Record<string, boolean> = {};
        channels.forEach((ch, index) => {
          const key = `${ch.type}-${index}`;
          if (prev[key]) next[key] = true;
        });
        return next;
      });
    }

    prevLenRef.current = nextLen;
  }, [channels]);

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 rounded-[var(--radius-lg)] bg-[var(--fill-tertiary)] py-14">
        <div className="spinner" />
        <p className="footnote">加载通道配置</p>
      </div>
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

  const canRemove = (type: string) =>
    channels.filter((c) => c.type === type).length > 1;

  const toggle = (key: string) => {
    setExpanded((prev) => ({ ...prev, [key]: !prev[key] }));
  };

  return (
    <div className="flex flex-col gap-4 pb-4">
      <div className="flex items-center justify-between gap-3">
        <div>
          <div className="subhead">推送通道</div>
          <p className="caption mt-0.5">测试仅发送到已启用通道</p>
        </div>
        <button
          type="button"
          className="btn"
          disabled={testing}
          onClick={onTest}
        >
          {testing ? "测试中…" : "测试"}
        </button>
      </div>

      <div className="flex flex-col gap-3">
        {channels.map((ch, index) => {
          const key = `${ch.type}-${index}`;
          const open = !!expanded[key];
          return (
            <section
              key={key}
              className="rounded-[var(--radius-lg)] bg-[var(--fill-tertiary)]"
            >
              <div className="flex items-center gap-2 px-3 py-3">
                <button
                  type="button"
                  className="pressable flex min-w-0 flex-1 items-center gap-2.5 text-left"
                  onClick={() => toggle(key)}
                  aria-expanded={open}
                >
                  <span
                    className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-[var(--bg-elevated)] text-[var(--label-secondary)] transition-transform duration-200"
                    style={{
                      transform: open ? "rotate(90deg)" : "rotate(0deg)",
                    }}
                  >
                    <IconChevron size={16} />
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="flex flex-wrap items-center gap-2">
                      <span className="rounded-full bg-[var(--bg-elevated)] px-2.5 py-1 text-[0.75rem] font-semibold tracking-wide">
                        {getChannelLabel(ch.type)}
                      </span>
                      <span
                        className="rounded-full px-2 py-0.5 text-[0.6875rem] font-semibold"
                        style={{
                          background: ch.enabled
                            ? "color-mix(in srgb, var(--green) 16%, transparent)"
                            : "var(--fill-secondary)",
                          color: ch.enabled
                            ? "var(--green)"
                            : "var(--label-tertiary)",
                        }}
                      >
                        {ch.enabled ? "已启用" : "未启用"}
                      </span>
                    </span>
                    <span className="caption truncate-safe mt-1 block">
                      {channelSummary(ch)}
                    </span>
                  </span>
                </button>

                <div
                  className="flex shrink-0 items-center gap-2"
                  onClick={(e) => e.stopPropagation()}
                >
                  <span className="hidden text-[0.75rem] font-medium text-[var(--label-tertiary)] sm:inline">
                    {ch.enabled ? "开" : "关"}
                  </span>
                  <Switch
                    checked={!!ch.enabled}
                    label={ch.enabled ? "关闭通道" : "启用通道"}
                    onChange={(next) =>
                      onChange(index, { ...ch, enabled: next })
                    }
                  />
                </div>
                {canRemove(ch.type) && (
                  <button
                    type="button"
                    className="btn btn-icon btn-ghost shrink-0"
                    aria-label="删除通道"
                    onClick={() => onRemove(index)}
                  >
                    <IconTrash size={16} />
                  </button>
                )}
              </div>

              {open && (
                <div className="flex flex-col gap-3 border-t border-[var(--hairline)] px-4 pt-3 pb-4">
                  <ChannelFields
                    ch={ch}
                    onChange={(next) => onChange(index, next)}
                  />
                </div>
              )}
            </section>
          );
        })}
      </div>

      <div className="rounded-[var(--radius-lg)] bg-[var(--fill-tertiary)] p-3">
        <select
          className="field mb-2"
          value={newChannelType}
          onChange={(e) => onNewChannelTypeChange(e.target.value)}
        >
          <option value="" disabled>
            选择通道类型…
          </option>
          {CHANNEL_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
        <button
          type="button"
          className="btn w-full"
          disabled={!newChannelType}
          onClick={onAdd}
        >
          <IconPlus size={16} />
          添加通道
        </button>
      </div>

      <button
        type="button"
        className="btn btn-primary sticky bottom-3 w-full py-3 shadow-[var(--shadow-float)]"
        disabled={saving}
        onClick={onSave}
      >
        {saving ? "保存中…" : "保存配置"}
      </button>
    </div>
  );
}
