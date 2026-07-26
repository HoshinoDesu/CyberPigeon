import type {
  AuthStatus,
  ChannelConfig,
  MessagesResponse,
  ModemInfo,
  Settings,
} from "./types";

export class AuthError extends Error {
  status: number;
  state: Partial<AuthStatus>;

  constructor(message: string, status: number, state: Partial<AuthStatus> = {}) {
    super(message);
    this.name = "AuthError";
    this.status = status;
    this.state = state;
  }
}

async function parseAuthFailure(res: Response): Promise<AuthError | null> {
  if (res.status !== 401 && res.status !== 428 && res.status !== 429) {
    return null;
  }
  try {
    const data = (await res.json()) as AuthStatus;
    return new AuthError(
      data.message ||
        (data.requires_setup ? "请先设置管理密码" : "登录已失效，请重新登录"),
      res.status,
      data,
    );
  } catch {
    return new AuthError("登录已失效，请重新登录", res.status);
  }
}

async function readErrorText(res: Response): Promise<string> {
  try {
    const text = await res.text();
    if (!text) return res.statusText || "请求失败";
    try {
      const json = JSON.parse(text) as { error?: string; message?: string };
      return json.error || json.message || text;
    } catch {
      return text;
    }
  } catch {
    return res.statusText || "请求失败";
  }
}

export async function apiFetch(url: string, options?: RequestInit): Promise<Response> {
  const res = await fetch(url, options);
  const authErr = await parseAuthFailure(res);
  if (authErr) throw authErr;
  return res;
}

export async function getAuthStatus(): Promise<AuthStatus> {
  const res = await fetch("/api/auth/status");
  if (!res.ok) throw new Error("状态检查失败");
  return res.json();
}

export async function login(password: string): Promise<void> {
  const res = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ password }),
  });
  const authErr = await parseAuthFailure(res);
  if (authErr) throw authErr;
  if (!res.ok) throw new Error(await readErrorText(res));
}

export async function setupPassword(password: string): Promise<void> {
  const res = await fetch("/api/auth/setup", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ password }),
  });
  const authErr = await parseAuthFailure(res);
  if (authErr) throw authErr;
  if (!res.ok) throw new Error(await readErrorText(res));
}

export async function logout(): Promise<void> {
  await fetch("/api/auth/logout", { method: "POST" });
}

export async function changePassword(
  currentPassword: string,
  newPassword: string,
): Promise<void> {
  const res = await apiFetch("/api/auth/change-password", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      current_password: currentPassword,
      new_password: newPassword,
    }),
  });
  if (!res.ok) throw new Error(await readErrorText(res));
}

export async function getModems(): Promise<ModemInfo[]> {
  const res = await apiFetch("/api/modems");
  if (!res.ok) throw new Error(await readErrorText(res));
  return res.json();
}

export async function getMessages(
  limit = 50,
  offset = 0,
): Promise<MessagesResponse> {
  const res = await apiFetch(`/api/messages?limit=${limit}&offset=${offset}`);
  if (!res.ok) throw new Error(await readErrorText(res));
  const data = await res.json();
  if (Array.isArray(data)) {
    return { items: data, total: data.length };
  }
  return {
    items: data.items || [],
    total: data.total || 0,
  };
}

export async function deleteMessage(id: string): Promise<void> {
  const res = await apiFetch("/api/messages/delete", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id }),
  });
  if (!res.ok) throw new Error(await readErrorText(res));
}

export async function deleteMessages(ids: string[]): Promise<number> {
  const res = await apiFetch("/api/messages/delete", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ ids }),
  });
  if (!res.ok) throw new Error(await readErrorText(res));
  const data = (await res.json()) as { deleted?: number };
  return data.deleted ?? ids.length;
}

export async function exportConfig(): Promise<void> {
  const res = await apiFetch("/api/config/export");
  if (!res.ok) throw new Error(await readErrorText(res));
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = "cyberpigeon-config.toml";
  a.click();
  URL.revokeObjectURL(url);
}

export async function importConfig(file: File): Promise<void> {
  const res = await apiFetch("/api/config/import", {
    method: "POST",
    headers: { "Content-Type": "application/toml" },
    body: file,
  });
  if (!res.ok) throw new Error(await readErrorText(res));
}

export async function getSettings(): Promise<Settings> {
  const res = await apiFetch("/api/settings");
  if (!res.ok) throw new Error(await readErrorText(res));
  return res.json();
}

export async function saveSettings(settings: Settings): Promise<void> {
  const res = await apiFetch("/api/settings/save", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(settings),
  });
  if (!res.ok) throw new Error(await readErrorText(res));
}

export async function getChannels(): Promise<ChannelConfig[]> {
  const res = await apiFetch("/api/channels");
  if (!res.ok) throw new Error(await readErrorText(res));
  const data = (await res.json()) as ChannelConfig[];
  return (data || []).map((ch) => {
    const next = { ...ch };
    if (next.type === "email" && !Array.isArray(next.to)) {
      next.to = [];
    }
    if (next.type === "webhook") {
      next.headers_json = next.headers
        ? JSON.stringify(next.headers, null, 2)
        : "";
    }
    return next;
  });
}

export function cleanChannelsForSave(channels: ChannelConfig[]): ChannelConfig[] {
  return channels.map((ch) => {
    const copy: ChannelConfig = { ...ch };
    if (copy.type === "webhook") {
      try {
        copy.headers = copy.headers_json
          ? (JSON.parse(copy.headers_json) as Record<string, string>)
          : {};
      } catch {
        copy.headers = {};
      }
      delete copy.headers_json;
    } else {
      delete copy.headers_json;
    }
    return copy;
  });
}

export async function saveChannels(channels: ChannelConfig[]): Promise<void> {
  const res = await apiFetch("/api/channels/save", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(cleanChannelsForSave(channels)),
  });
  if (!res.ok) throw new Error(await readErrorText(res));
}

export async function testChannels(channels: ChannelConfig[]): Promise<{
  success: boolean;
  message?: string;
}> {
  const res = await apiFetch("/api/channels/test", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(cleanChannelsForSave(channels)),
  });
  if (!res.ok) throw new Error(await readErrorText(res));
  return res.json();
}

export async function runUssd(
  imei: string,
  code: string,
): Promise<{ reply: string }> {
  const res = await apiFetch("/api/ussd", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ imei, code }),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(
      (data as { error?: string }).error || (await readErrorText(res)),
    );
  }
  return data as { reply: string };
}

export function validateManagementPassword(
  value: string,
  label = "密码",
): string {
  const trimmed = (value || "").trim();
  if (trimmed.length < 6) {
    throw new Error(`${label}至少需要 6 位`);
  }
  if (trimmed.length > 256) {
    throw new Error(`${label}不能超过 256 位`);
  }
  if (/[\x00-\x1F\x7F]/.test(trimmed)) {
    throw new Error(`${label}不能包含控制字符`);
  }
  return trimmed;
}

export function validateUssdCode(code: string): string {
  if (!code) return "";
  for (const c of code) {
    if (c !== "*" && c !== "#" && (c < "0" || c > "9")) {
      return "仅允许数字、* 和 # 字符";
    }
  }
  return "";
}
