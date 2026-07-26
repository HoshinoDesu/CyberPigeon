export type AuthStatus = {
  auth_enabled: boolean;
  requires_setup: boolean;
  authenticated: boolean;
  message?: string;
};

export type ModemInfo = {
  imei: string;
  model: string;
  manufacturer: string;
  number: string;
  signal_quality: number;
  operator_name: string;
  iccid: string;
  state: string;
  display_name: string;
};

export type Message = {
  id: string;
  modem: string;
  number: string;
  text: string;
  timestamp: string;
  saved?: string;
};

export type MessagesResponse = {
  items: Message[];
  total: number;
};

export type ModemNameEntry = {
  imei: string;
  name: string;
};

export type ForwardRulesMode = "off" | "blacklist" | "whitelist";

export type ForwardRules = {
  mode: ForwardRulesMode;
  keywords: string[];
  senders: string[];
};

export type Settings = {
  device_name: string;
  device_name_in_title: boolean;
  device_name_in_body: boolean;
  always_on_modems: boolean;
  modems: ModemNameEntry[];
  forward_rules: ForwardRules;
};

export type ChannelType =
  | "email"
  | "bark"
  | "gotify"
  | "serverchan"
  | "webhook"
  | "wecom"
  | "feishu"
  | "dingtalk"
  | "telegram";

export type ChannelConfig = {
  type: ChannelType | string;
  enabled: boolean;
  request_timeout_sec?: number;
  allow_private_network?: boolean;

  // email
  host?: string;
  port?: number;
  username?: string;
  password?: string;
  from?: string;
  to?: string[];
  use_tls?: boolean;

  // bark / gotify / feishu / dingtalk
  endpoint?: string;
  title?: string;

  // gotify
  token?: string;
  priority?: number;

  // serverchan
  send_key?: string;

  // webhook
  url?: string;
  method?: string;
  headers?: Record<string, string>;
  /** frontend-only helper for editing headers as JSON text */
  headers_json?: string;

  // wecom
  corp_id?: string;
  corp_secret?: string;
  agent_id?: number;
  to_user?: string;

  // feishu
  app_id?: string;
  app_secret?: string;
  receive_id?: string;
  receive_id_type?: string;

  // dingtalk
  webhook_url?: string;
  sign_secret?: string;

  // telegram
  bot_token?: string;
  chat_id?: string;
  api_url?: string;
};

export type UssdState = {
  code: string;
  result: string;
  loading: boolean;
  error: string;
};

export const EMPTY_USSD_STATE: UssdState = {
  code: "",
  result: "",
  loading: false,
  error: "",
};

export type ToastItem = {
  id: number;
  message: string;
  type: "success" | "error" | "info" | "warning";
  leaving: boolean;
};

export type ThemeMode = "dark" | "light" | "auto";

export type WsNewMessage = {
  type: "new_message";
  message: Message;
};
