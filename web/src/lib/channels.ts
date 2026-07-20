import type { ChannelConfig, ChannelType } from "./types";

export const CHANNEL_LABELS: Record<string, string> = {
  email: "邮件推送 (Email)",
  bark: "Bark 推送",
  gotify: "Gotify 推送",
  serverchan: "Server酱3 推送",
  webhook: "Webhook 推送",
  wecom: "企业微信推送",
  feishu: "飞书推送",
  dingtalk: "钉钉推送",
  telegram: "Telegram 推送",
};

export const CHANNEL_OPTIONS: { value: ChannelType; label: string }[] = [
  { value: "email", label: "Email 邮件推送" },
  { value: "bark", label: "Bark 推送" },
  { value: "gotify", label: "Gotify 推送" },
  { value: "serverchan", label: "Server酱3 推送" },
  { value: "webhook", label: "Webhook 推送" },
  { value: "wecom", label: "企业微信推送" },
  { value: "feishu", label: "飞书推送" },
  { value: "dingtalk", label: "钉钉推送" },
  { value: "telegram", label: "Telegram 推送" },
];

export function getChannelLabel(type: string): string {
  return CHANNEL_LABELS[type] || type || "未知";
}

export function createChannelDefault(type: ChannelType): ChannelConfig {
  const defaults: Record<ChannelType, ChannelConfig> = {
    email: {
      type: "email",
      enabled: false,
      request_timeout_sec: 10,
      port: 587,
      use_tls: true,
      to: [],
    },
    bark: { type: "bark", enabled: false, request_timeout_sec: 10 },
    gotify: {
      type: "gotify",
      enabled: false,
      request_timeout_sec: 10,
      priority: 5,
    },
    serverchan: {
      type: "serverchan",
      enabled: false,
      request_timeout_sec: 10,
    },
    webhook: {
      type: "webhook",
      enabled: false,
      request_timeout_sec: 10,
      method: "POST",
      headers_json: "",
      allow_private_network: false,
    },
    wecom: { type: "wecom", enabled: false, request_timeout_sec: 10 },
    feishu: {
      type: "feishu",
      enabled: false,
      request_timeout_sec: 10,
      receive_id_type: "open_id",
    },
    dingtalk: { type: "dingtalk", enabled: false, request_timeout_sec: 10 },
    telegram: {
      type: "telegram",
      enabled: false,
      request_timeout_sec: 10,
      api_url: "https://api.telegram.org",
    },
  };
  return structuredClone(defaults[type]);
}

export function toStr(arr?: string[]): string {
  return Array.isArray(arr) ? arr.join(", ") : "";
}

export function fromStr(str: string): string[] {
  return str
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
}
