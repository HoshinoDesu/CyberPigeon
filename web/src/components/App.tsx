"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  AuthError,
  changePassword,
  getAuthStatus,
  getChannels,
  getMessages,
  getModems,
  getSettings,
  login,
  logout,
  runUssd,
  saveChannels,
  saveSettings,
  setupPassword,
  testChannels,
  validateManagementPassword,
  validateUssdCode,
  deleteMessage,
} from "@/lib/api";
import { createChannelDefault, getChannelLabel } from "@/lib/channels";
import { formatClockTime } from "@/lib/format";
import type {
  ChannelConfig,
  ChannelType,
  Message,
  ModemInfo,
  Settings,
  ThemeMode,
  ToastItem,
  UssdState,
  WsNewMessage,
} from "@/lib/types";
import { AuthGate } from "./AuthGate";
import { ChannelsPanel } from "./ChannelsPanel";
import { ConfirmModal } from "./ConfirmModal";
import { DeviceOverview } from "./DeviceOverview";
import { DevicesPanel } from "./DevicesPanel";
import {
  IconBell,
  IconDevice,
  IconInbox,
  IconLogout,
  IconMonitor,
  IconMoon,
  IconShield,
  IconSun,
} from "./icons";
import { MessageList } from "./MessageList";
import { SecurityPanel } from "./SecurityPanel";
import { ToastStack } from "./ToastStack";

const MESSAGE_LIMIT = 50;

type NavKey = "messages" | "devices" | "channels" | "security";

const NAV_ITEMS: {
  key: NavKey;
  label: string;
  Icon: typeof IconInbox;
}[] = [
  { key: "messages", label: "消息", Icon: IconInbox },
  { key: "devices", label: "设备", Icon: IconDevice },
  { key: "channels", label: "推送", Icon: IconBell },
  { key: "security", label: "安全", Icon: IconShield },
];

const PAGE_META: Record<NavKey, { title: string; subtitle: string }> = {
  messages: { title: "消息", subtitle: "收件箱与实时推送" },
  devices: { title: "设备", subtitle: "调制解调器、信号与 USSD" },
  channels: { title: "推送", subtitle: "通知通道配置与测试" },
  security: { title: "安全", subtitle: "管理密码" },
};

function applyTheme(theme: ThemeMode) {
  const root = document.documentElement;
  if (theme === "auto") {
    const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
    root.setAttribute("data-theme", prefersDark ? "dark" : "light");
  } else {
    root.setAttribute("data-theme", theme);
  }
}

export function App() {
  const [authLoading, setAuthLoading] = useState(true);
  const [authEnabled, setAuthEnabled] = useState(false);
  const [authenticated, setAuthenticated] = useState(false);
  const [requiresSetup, setRequiresSetup] = useState(false);
  const [authSubmitting, setAuthSubmitting] = useState(false);
  const [authPassword, setAuthPassword] = useState("");
  const [authPasswordConfirm, setAuthPasswordConfirm] = useState("");
  const [authError, setAuthError] = useState("");
  const [authMessage, setAuthMessage] = useState("");

  const [modems, setModems] = useState<ModemInfo[]>([]);
  const [messages, setMessages] = useState<Message[]>([]);
  const [totalMessages, setTotalMessages] = useState(0);
  const [messageOffset, setMessageOffset] = useState(0);
  const [channels, setChannels] = useState<ChannelConfig[]>([]);
  const [settings, setSettings] = useState<Settings>({
    device_name: "",
    device_name_in_title: false,
    device_name_in_body: false,
    always_on_modems: false,
    modems: [],
  });

  const [modemsLoading, setModemsLoading] = useState(true);
  const [messagesLoading, setMessagesLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [channelsLoading, setChannelsLoading] = useState(false);
  const [modemsError, setModemsError] = useState<string | null>(null);
  const [messagesError, setMessagesError] = useState<string | null>(null);
  const [channelsError, setChannelsError] = useState<string | null>(null);
  const [settingsSaving, setSettingsSaving] = useState(false);
  const [channelsSaving, setChannelsSaving] = useState(false);
  const [testingChannels, setTestingChannels] = useState(false);

  const [passwordForm, setPasswordForm] = useState({
    current_password: "",
    new_password: "",
    confirm_password: "",
  });
  const [passwordChanging, setPasswordChanging] = useState(false);
  const [passwordChangeError, setPasswordChangeError] = useState("");

  const [lastUpdate, setLastUpdate] = useState("");
  const [page, setPage] = useState<NavKey>("messages");
  const [theme, setTheme] = useState<ThemeMode>("auto");
  const [searchQuery, setSearchQuery] = useState("");
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [messageToDelete, setMessageToDelete] = useState<Message | null>(null);
  const [wsConnected, setWsConnected] = useState(false);
  const [newChannelType, setNewChannelType] = useState("");
  const [ussdStates, setUssdStates] = useState<Record<string, UssdState>>({});
  const [toasts, setToasts] = useState<ToastItem[]>([]);

  const toastIdRef = useRef(0);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const refreshTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const authenticatedRef = useRef(false);
  const authEnabledRef = useRef(false);
  const channelsLoadedRef = useRef(false);

  useEffect(() => {
    authenticatedRef.current = authenticated;
  }, [authenticated]);

  useEffect(() => {
    authEnabledRef.current = authEnabled;
  }, [authEnabled]);

  const showToast = useCallback(
    (message: string, type: ToastItem["type"] = "info", duration = 3000) => {
      const id = ++toastIdRef.current;
      setToasts((prev) => [...prev, { id, message, type, leaving: false }]);
      window.setTimeout(() => {
        setToasts((prev) =>
          prev.map((t) => (t.id === id ? { ...t, leaving: true } : t)),
        );
        window.setTimeout(() => {
          setToasts((prev) => prev.filter((t) => t.id !== id));
        }, 220);
      }, duration);
    },
    [],
  );

  const closeWebSocket = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
    if (wsRef.current) {
      wsRef.current.onclose = null;
      wsRef.current.close();
      wsRef.current = null;
    }
    setWsConnected(false);
  }, []);

  const handleAuthFailure = useCallback(
    (err: unknown) => {
      if (!(err instanceof AuthError)) return false;
      if (typeof err.state.auth_enabled === "boolean") {
        setAuthEnabled(err.state.auth_enabled);
      } else {
        setAuthEnabled(true);
      }
      setRequiresSetup(!!err.state.requires_setup);
      setAuthenticated(!!err.state.authenticated);
      setAuthMessage(err.message || err.state.message || "");
      closeWebSocket();
      return true;
    },
    [closeWebSocket],
  );

  const updateLastUpdateTime = useCallback(() => {
    setLastUpdate(`更新于 ${formatClockTime()}`);
  }, []);

  const loadModems = useCallback(async () => {
    try {
      setModemsError(null);
      const data = await getModems();
      setModems(data);
      updateLastUpdateTime();
    } catch (e) {
      if (handleAuthFailure(e)) return;
      setModemsError(`加载设备信息失败: ${(e as Error).message}`);
    } finally {
      setModemsLoading(false);
    }
  }, [handleAuthFailure, updateLastUpdateTime]);

  const messageOffsetRef = useRef(0);
  useEffect(() => {
    messageOffsetRef.current = messageOffset;
  }, [messageOffset]);

  const loadMessages = useCallback(
    async (append = false) => {
      try {
        if (append) {
          setLoadingMore(true);
        } else {
          setMessagesLoading(true);
          setMessageOffset(0);
          messageOffsetRef.current = 0;
          setMessagesError(null);
        }
        const offset = append ? messageOffsetRef.current : 0;
        const data = await getMessages(MESSAGE_LIMIT, offset);
        if (append) {
          setMessages((prev) => [...prev, ...data.items]);
        } else {
          setMessages(data.items);
        }
        setTotalMessages(data.total);
        const nextOffset = (append ? messageOffsetRef.current : 0) + data.items.length;
        messageOffsetRef.current = nextOffset;
        setMessageOffset(nextOffset);
      } catch (e) {
        if (handleAuthFailure(e)) return;
        if (!append) {
          setMessagesError(`加载短信失败: ${(e as Error).message}`);
        } else {
          showToast(`加载更多失败: ${(e as Error).message}`, "error");
        }
      } finally {
        setMessagesLoading(false);
        setLoadingMore(false);
      }
    },
    [handleAuthFailure, showToast],
  );

  const loadSettings = useCallback(async () => {
    try {
      const data = await getSettings();
      setSettings({
        device_name: data.device_name || "",
        device_name_in_title: !!data.device_name_in_title,
        device_name_in_body: !!data.device_name_in_body,
        always_on_modems: !!data.always_on_modems,
        modems: Array.isArray(data.modems) ? data.modems : [],
      });
    } catch (e) {
      if (handleAuthFailure(e)) return;
      showToast(`加载设置失败: ${(e as Error).message}`, "error");
    }
  }, [handleAuthFailure, showToast]);

  const loadChannels = useCallback(async () => {
    try {
      setChannelsLoading(true);
      setChannelsError(null);
      const data = await getChannels();
      setChannels(data);
      channelsLoadedRef.current = true;
    } catch (e) {
      if (handleAuthFailure(e)) return;
      setChannelsError(`加载通道配置失败: ${(e as Error).message}`);
    } finally {
      setChannelsLoading(false);
    }
  }, [handleAuthFailure]);

  const connectWebSocket = useCallback(() => {
    if (authEnabledRef.current && !authenticatedRef.current) return;

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${window.location.host}/ws`;
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      setWsConnected(true);
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data as string) as WsNewMessage;
        if (data.type === "new_message" && data.message) {
          setMessages((prev) => [data.message, ...prev]);
          setTotalMessages((prev) => prev + 1);
          setMessageOffset((prev) => prev + 1);
          updateLastUpdateTime();
        }
      } catch {
        // ignore malformed payloads
      }
    };

    ws.onerror = () => {
      setWsConnected(false);
    };

    ws.onclose = () => {
      setWsConnected(false);
      if (authEnabledRef.current && !authenticatedRef.current) return;
      reconnectTimerRef.current = setTimeout(() => {
        connectWebSocket();
      }, 5000);
    };
  }, [updateLastUpdateTime]);

  const initializeApp = useCallback(async () => {
    await Promise.all([loadModems(), loadMessages(false), loadSettings()]);
    connectWebSocket();
  }, [connectWebSocket, loadMessages, loadModems, loadSettings]);

  const checkAuth = useCallback(async () => {
    try {
      setAuthLoading(true);
      setAuthError("");
      const data = await getAuthStatus();
      setAuthEnabled(!!data.auth_enabled);
      setRequiresSetup(!!data.requires_setup);
      setAuthenticated(!!data.authenticated || !data.auth_enabled);
      setAuthMessage(data.message || "");
      return !!data.authenticated || !data.auth_enabled;
    } catch (e) {
      setAuthError(`无法获取登录状态: ${(e as Error).message}`);
      setAuthenticated(false);
      return false;
    } finally {
      setAuthLoading(false);
    }
  }, []);

  useEffect(() => {
    const saved = (localStorage.getItem("theme") as ThemeMode | null) || "dark";
    const initial: ThemeMode =
      saved === "light" || saved === "dark" || saved === "auto" ? saved : "dark";
    setTheme(initial);
    applyTheme(initial);

    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => {
      if ((localStorage.getItem("theme") || "dark") === "auto") {
        applyTheme("auto");
      }
    };
    mq.addEventListener("change", onChange);

    void (async () => {
      const ok = await checkAuth();
      if (ok) {
        await initializeApp();
      }
    })();

    refreshTimerRef.current = setInterval(() => {
      if (authenticatedRef.current || !authEnabledRef.current) {
        void loadModems();
      }
    }, 30_000);

    return () => {
      mq.removeEventListener("change", onChange);
      if (refreshTimerRef.current) clearInterval(refreshTimerRef.current);
      closeWebSocket();
    };
    // mount-only bootstrap
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const filteredMessages = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    if (!q) return messages;
    return messages.filter(
      (msg) =>
        (msg.text && msg.text.toLowerCase().includes(q)) ||
        (msg.number && msg.number.includes(q)) ||
        (msg.modem && msg.modem.toLowerCase().includes(q)),
    );
  }, [messages, searchQuery]);

  const hasMoreMessages = messages.length < totalMessages;

  const modemNames = useMemo(() => {
    const map: Record<string, string> = {};
    for (const m of settings.modems) {
      if (m.imei && m.name) map[m.imei] = m.name;
    }
    return map;
  }, [settings.modems]);

  const toggleTheme = () => {
    const themes: ThemeMode[] = ["dark", "light", "auto"];
    const idx = (themes.indexOf(theme) + 1) % themes.length;
    const next = themes[idx];
    setTheme(next);
    localStorage.setItem("theme", next);
    applyTheme(next);
  };

  const onLogin = async () => {
    try {
      setAuthSubmitting(true);
      setAuthError("");
      const password = validateManagementPassword(authPassword);
      await login(password);
      setAuthenticated(true);
      setAuthMessage("");
      setAuthPassword("");
      await initializeApp();
    } catch (e) {
      if (handleAuthFailure(e)) {
        setAuthError((e as Error).message);
      } else {
        setAuthError((e as Error).message);
      }
    } finally {
      setAuthSubmitting(false);
    }
  };

  const onSetup = async () => {
    try {
      setAuthSubmitting(true);
      setAuthError("");
      const password = validateManagementPassword(authPassword);
      const confirm = validateManagementPassword(authPasswordConfirm, "确认密码");
      if (password !== confirm) throw new Error("两次输入的密码不一致");
      await setupPassword(password);
      setRequiresSetup(false);
      setAuthenticated(true);
      setAuthMessage("");
      setAuthPassword("");
      setAuthPasswordConfirm("");
      await initializeApp();
    } catch (e) {
      setAuthError((e as Error).message);
    } finally {
      setAuthSubmitting(false);
    }
  };

  const onLogout = async () => {
    try {
      await logout();
    } finally {
      setAuthenticated(false);
      setAuthMessage("你已退出登录，请重新登录后继续操作");
      setAuthPassword("");
      setAuthPasswordConfirm("");
      closeWebSocket();
    }
  };

  const onChangePassword = async () => {
    try {
      setPasswordChanging(true);
      setPasswordChangeError("");
      const current = (passwordForm.current_password || "").trim();
      if (!current) throw new Error("请输入当前密码");
      const next = validateManagementPassword(passwordForm.new_password, "新密码");
      const confirm = validateManagementPassword(
        passwordForm.confirm_password,
        "确认新密码",
      );
      if (next !== confirm) throw new Error("两次输入的新密码不一致");
      await changePassword(current, next);
      setPasswordForm({
        current_password: "",
        new_password: "",
        confirm_password: "",
      });
      showToast("密码已更新，其他设备需要重新登录", "success", 5000);
    } catch (e) {
      if (handleAuthFailure(e)) return;
      setPasswordChangeError((e as Error).message);
    } finally {
      setPasswordChanging(false);
    }
  };

  const onSaveSettings = async () => {
    try {
      setSettingsSaving(true);
      await saveSettings(settings);
      showToast("设置已保存成功", "success");
    } catch (e) {
      if (handleAuthFailure(e)) return;
      showToast(`保存设置失败: ${(e as Error).message}`, "error", 5000);
    } finally {
      setSettingsSaving(false);
    }
  };

  const onSaveChannels = async () => {
    try {
      setChannelsSaving(true);
      await saveChannels(channels);
      showToast("配置已保存成功", "success");
    } catch (e) {
      if (handleAuthFailure(e)) return;
      showToast(`保存失败: ${(e as Error).message}`, "error", 5000);
    } finally {
      setChannelsSaving(false);
    }
  };

  const onTestChannels = async () => {
    const enabled = channels.filter((ch) => ch.enabled);
    if (enabled.length === 0) {
      showToast("没有启用的推送通道，请先启用至少一个通道", "warning");
      return;
    }
    try {
      setTestingChannels(true);
      const result = await testChannels(enabled);
      if (result.success) {
        showToast(
          `测试推送已发送到 ${enabled.length} 个通道，请检查是否收到通知`,
          "success",
          5000,
        );
      } else {
        showToast("测试失败: 未知错误", "error", 5000);
      }
    } catch (e) {
      if (handleAuthFailure(e)) return;
      showToast(`测试失败: ${(e as Error).message}`, "error", 5000);
    } finally {
      setTestingChannels(false);
    }
  };

  const onAddChannel = () => {
    if (!newChannelType) return;
    const type = newChannelType as ChannelType;
    setChannels((prev) => [...prev, createChannelDefault(type)]);
    setNewChannelType("");
    showToast(`已添加 ${getChannelLabel(type)} 通道`, "success");
  };

  const onRemoveChannel = (index: number) => {
    const ch = channels[index];
    setChannels((prev) => prev.filter((_, i) => i !== index));
    showToast(`已移除 ${getChannelLabel(ch.type)} 通道`, "info");
  };

  const onUssdCodeChange = (imei: string, code: string) => {
    const error = validateUssdCode(code);
    setUssdStates((prev) => ({
      ...prev,
      [imei]: {
        code,
        result: prev[imei]?.result || "",
        loading: prev[imei]?.loading || false,
        error,
      },
    }));
  };

  const onRunUssd = async (modem: ModemInfo) => {
    const state = ussdStates[modem.imei] || {
      code: "",
      result: "",
      loading: false,
      error: "",
    };
    if (!state.code) {
      showToast("请输入 USSD 代码", "warning");
      return;
    }
    if (state.error) return;

    setUssdStates((prev) => ({
      ...prev,
      [modem.imei]: { ...state, loading: true, result: "" },
    }));

    try {
      const data = await runUssd(modem.imei, state.code);
      setUssdStates((prev) => ({
        ...prev,
        [modem.imei]: {
          ...prev[modem.imei],
          loading: false,
          result:
            data.reply ||
            "USSD 请求已发送，但未收到即时回复（请留意短信通知）。",
        },
      }));
    } catch (e) {
      if (handleAuthFailure(e)) return;
      setUssdStates((prev) => ({
        ...prev,
        [modem.imei]: { ...prev[modem.imei], loading: false },
      }));
      showToast(`USSD 执行失败: ${(e as Error).message}`, "error");
    }
  };

  const confirmDelete = (msg: Message) => {
    setMessageToDelete(msg);
    setShowDeleteModal(true);
  };

  const cancelDelete = () => {
    setShowDeleteModal(false);
    setMessageToDelete(null);
  };

  const doDelete = async () => {
    if (!messageToDelete) return;
    try {
      await deleteMessage(messageToDelete.id);
      setMessages((prev) => prev.filter((m) => m.id !== messageToDelete.id));
      setTotalMessages((prev) => Math.max(0, prev - 1));
      cancelDelete();
      showToast("短信已删除", "success");
    } catch (e) {
      if (handleAuthFailure(e)) return;
      showToast(`删除失败: ${(e as Error).message}`, "error");
    }
  };

  const switchPage = (key: NavKey) => {
    setPage(key);
    if (key === "channels" && !channelsLoadedRef.current && !channelsLoading) {
      void loadChannels();
    }
    if (key === "devices") {
      void loadModems();
    }
  };

  const statusText = wsConnected
    ? lastUpdate || "实时连接"
    : "已断开，重连中…";

  const ThemeIcon =
    theme === "light" ? IconSun : theme === "dark" ? IconMoon : IconMonitor;

  const meta = PAGE_META[page];

  if (authLoading) {
    return (
      <div className="flex min-h-dvh flex-col items-center justify-center gap-3">
        <div className="spinner" />
        <p className="footnote">正在检查登录状态</p>
      </div>
    );
  }

  if (authEnabled && !authenticated) {
    return (
      <>
        <AuthGate
          requiresSetup={requiresSetup}
          message={authMessage}
          error={authError}
          password={authPassword}
          passwordConfirm={authPasswordConfirm}
          submitting={authSubmitting}
          onPasswordChange={setAuthPassword}
          onPasswordConfirmChange={setAuthPasswordConfirm}
          onSubmit={requiresSetup ? onSetup : onLogin}
        />
        <ToastStack toasts={toasts} />
      </>
    );
  }

  return (
    <div className="app-shell">
      <aside className="app-rail">
        <div className="mb-6 px-2">
          <div className="text-[1.05rem] font-semibold tracking-[-0.03em]">
            CyberPigeon
          </div>
          <div className="mt-1.5 flex items-center gap-1.5">
            <span className="status-live" data-off={!wsConnected} />
            <span className="caption truncate">{statusText}</span>
          </div>
        </div>

        <nav className="flex flex-1 flex-col gap-1">
          {NAV_ITEMS.map(({ key, label, Icon }) => (
            <button
              key={key}
              type="button"
              className="rail-nav-item"
              data-active={page === key}
              onClick={() => switchPage(key)}
            >
              <Icon size={18} />
              <span className="flex-1">{label}</span>
              {key === "devices" && (
                <span className="caption mono">{modems.length}</span>
              )}
              {key === "messages" && totalMessages > 0 && (
                <span className="caption mono">{totalMessages}</span>
              )}
            </button>
          ))}
        </nav>

        <div className="mt-4 px-0.5">
          <DeviceOverview
            modems={modems}
            loading={modemsLoading}
            onOpenDevices={() => switchPage("devices")}
          />
        </div>

        <div className="mt-3 flex flex-col gap-1 border-t border-[var(--hairline)] pt-3">
          <button
            type="button"
            className="rail-nav-item"
            onClick={toggleTheme}
          >
            <ThemeIcon size={18} />
            <span>
              {theme === "light" ? "浅色" : theme === "dark" ? "深色" : "跟随系统"}
            </span>
          </button>
          {authEnabled && authenticated && (
            <button
              type="button"
              className="rail-nav-item"
              onClick={() => void onLogout()}
            >
              <IconLogout size={18} />
              <span>退出登录</span>
            </button>
          )}
        </div>
      </aside>

      <div className="app-main">
        <header className="app-top">
          <div className="min-w-0">
            <div className="truncate text-[1.05rem] font-semibold tracking-[-0.025em]">
              {meta.title}
            </div>
            <div className="flex items-center gap-1.5 md:hidden">
              <span className="status-live" data-off={!wsConnected} />
              <span className="caption truncate">{statusText}</span>
            </div>
            <p className="caption hidden truncate md:block">{meta.subtitle}</p>
          </div>
          <div className="flex min-w-0 items-center gap-2">
            <div className="hidden min-w-[13rem] max-w-[18rem] sm:block lg:hidden">
              <DeviceOverview
                modems={modems}
                loading={modemsLoading}
                compact
                onOpenDevices={() => switchPage("devices")}
              />
            </div>
            <button
              type="button"
              className="btn btn-icon btn-ghost md:hidden"
              onClick={toggleTheme}
              aria-label="切换主题"
            >
              <ThemeIcon size={18} />
            </button>
            {authEnabled && authenticated && (
              <button
                type="button"
                className="btn btn-icon btn-ghost md:hidden"
                onClick={() => void onLogout()}
                aria-label="退出登录"
              >
                <IconLogout size={18} />
              </button>
            )}
          </div>
        </header>

        <main className="app-content">
          {page === "messages" && (
            <MessageList
              messages={messages}
              filtered={filteredMessages}
              loading={messagesLoading}
              error={messagesError}
              searchQuery={searchQuery}
              onSearchChange={setSearchQuery}
              hasMore={hasMoreMessages}
              loadingMore={loadingMore}
              total={totalMessages}
              modemNames={modemNames}
              onLoadMore={() => {
                if (!hasMoreMessages || loadingMore) return;
                void loadMessages(true);
              }}
              onDelete={confirmDelete}
            />
          )}

          {page === "devices" && (
            <div className="mx-auto max-w-[40rem]">
              <div className="mb-5">
                <h1 className="display">设备</h1>
                <p className="footnote mt-1">信号、本机名与 USSD</p>
              </div>
              <DevicesPanel
                modems={modems}
                loading={modemsLoading}
                error={modemsError}
                ussdStates={ussdStates}
                onUssdCodeChange={onUssdCodeChange}
                onRunUssd={(m) => void onRunUssd(m)}
                settings={settings}
                settingsSaving={settingsSaving}
                onSettingsChange={setSettings}
                onSaveSettings={() => void onSaveSettings()}
              />
            </div>
          )}

          {page === "channels" && (
            <div className="mx-auto max-w-[40rem]">
              <div className="mb-5">
                <h1 className="display">推送</h1>
                <p className="footnote mt-1">配置并测试通知通道</p>
              </div>
              <ChannelsPanel
                channels={channels}
                loading={channelsLoading}
                error={channelsError}
                saving={channelsSaving}
                testing={testingChannels}
                newChannelType={newChannelType}
                onNewChannelTypeChange={setNewChannelType}
                onChange={(index, next) =>
                  setChannels((prev) =>
                    prev.map((item, i) => (i === index ? next : item)),
                  )
                }
                onAdd={onAddChannel}
                onRemove={onRemoveChannel}
                onSave={() => void onSaveChannels()}
                onTest={() => void onTestChannels()}
              />
            </div>
          )}

          {page === "security" && (
            <div className="mx-auto max-w-[28rem]">
              <div className="mb-5">
                <h1 className="display">安全</h1>
                <p className="footnote mt-1">管理控制台密码</p>
              </div>
              <SecurityPanel
                currentPassword={passwordForm.current_password}
                newPassword={passwordForm.new_password}
                confirmPassword={passwordForm.confirm_password}
                error={passwordChangeError}
                changing={passwordChanging}
                onChange={(field, value) =>
                  setPasswordForm((prev) => ({ ...prev, [field]: value }))
                }
                onSubmit={() => void onChangePassword()}
              />
            </div>
          )}
        </main>
      </div>

      <nav className="app-tabbar" aria-label="主导航">
        {NAV_ITEMS.map(({ key, label, Icon }) => (
          <button
            key={key}
            type="button"
            className="tab-item"
            data-active={page === key}
            onClick={() => switchPage(key)}
          >
            <Icon size={20} />
            {label}
          </button>
        ))}
      </nav>

      <ConfirmModal
        open={showDeleteModal}
        title="删除这条短信？"
        danger
        confirmLabel="删除"
        onCancel={cancelDelete}
        onConfirm={() => void doDelete()}
        message={
          <>
            来自{" "}
            <span className="font-semibold text-[var(--label)]">
              {messageToDelete?.number}
            </span>{" "}
            的记录将被永久移除。
          </>
        }
      />

      <ToastStack toasts={toasts} />
    </div>
  );
}
