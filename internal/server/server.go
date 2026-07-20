package server

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/HoshinoDesu/CyberPigeon/internal/config"
	"github.com/HoshinoDesu/CyberPigeon/internal/forwarder"
	"github.com/HoshinoDesu/CyberPigeon/internal/modem"
	"github.com/HoshinoDesu/CyberPigeon/internal/notifier"
	"github.com/HoshinoDesu/CyberPigeon/internal/storage"
	"github.com/gorilla/websocket"
)

//go:embed all:web
var webFiles embed.FS

// wsClient 封装 WebSocket 连接及其独立写锁
type wsClient struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
}

// Server Web 服务器
type Server struct {
	cfg         *config.Config
	cfgMu       sync.RWMutex
	forwarder   *forwarder.Forwarder
	storage     *storage.Storage
	server      *http.Server
	configPath  string
	clients     map[*wsClient]bool
	clientsMu   sync.RWMutex
	upgrader    websocket.Upgrader
	authLimiter *authRateLimiter
}

// New 创建服务器
func New(cfg *config.Config, fwd *forwarder.Forwarder, store *storage.Storage, configPath string) *Server {
	serverCfg := cfg.Clone()
	if store == nil && serverCfg.Server.Enabled {
		originalListen := serverCfg.Server.Listen
		serverCfg.Server.Listen = localOnlyListenAddr(originalListen)
		if serverCfg.Server.Listen != originalListen {
			slog.Warn("存储未启用，Web 管理认证不可用，已强制仅监听本机地址", "from", originalListen, "listen", serverCfg.Server.Listen)
		}
	}
	return &Server{
		cfg:         serverCfg,
		forwarder:   fwd,
		storage:     store,
		configPath:  configPath,
		clients:     make(map[*wsClient]bool),
		authLimiter: newAuthRateLimiter(5, 15*time.Minute),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return isOriginAllowed(r, serverCfg.Server.AllowedOrigins) },
		},
	}
}

func localOnlyListenAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "127.0.0.1:8080"
	}
	if _, err := strconv.Atoi(addr); err == nil {
		return net.JoinHostPort("127.0.0.1", addr)
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		if strings.HasPrefix(addr, ":") {
			return "127.0.0.1" + addr
		}
		return net.JoinHostPort("127.0.0.1", "8080")
	}
	if isLoopbackListenHost(host) {
		return addr
	}
	return net.JoinHostPort("127.0.0.1", port)
}

func isLoopbackListenHost(host string) bool {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isOriginAllowed(r *http.Request, allowedOrigins []string) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}

	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}

	if len(allowedOrigins) == 0 {
		return strings.EqualFold(u.Host, r.Host)
	}

	for _, allowed := range allowedOrigins {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" {
			continue
		}
		if allowed == "*" || strings.EqualFold(allowed, origin) || strings.EqualFold(allowed, u.Host) {
			return true
		}
	}

	return false
}

// Run 运行服务器
func (s *Server) Run(ctx context.Context) error {
	s.cfgMu.RLock()
	serverEnabled := s.cfg.Server.Enabled
	listenAddr := s.cfg.Server.Listen
	s.cfgMu.RUnlock()
	if !serverEnabled {
		<-ctx.Done()
		return nil
	}

	mux := http.NewServeMux()

	// API 路由
	mux.HandleFunc("/api/auth/status", s.handleAuthStatus)
	mux.HandleFunc("/api/auth/setup", s.handleAuthSetup)
	mux.HandleFunc("/api/auth/login", s.handleAuthLogin)
	mux.HandleFunc("/api/auth/logout", s.handleAuthLogout)
	mux.HandleFunc("/api/auth/change-password", s.handleAuthChangePassword)
	mux.HandleFunc("/api/modems", s.withAPIAuth(s.handleModems))
	mux.HandleFunc("/api/messages", s.withAPIAuth(s.handleMessages))
	mux.HandleFunc("/api/messages/delete", s.withAPIAuth(s.handleDeleteMessage))
	mux.HandleFunc("/api/channels", s.withAPIAuth(s.handleChannels))
	mux.HandleFunc("/api/channels/save", s.withAPIAuth(s.handleSaveChannels))
	mux.HandleFunc("/api/channels/test", s.withAPIAuth(s.handleTestChannel))
	mux.HandleFunc("/api/ussd", s.withAPIAuth(s.handleUSSD))
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/api/settings", s.withAPIAuth(s.handleSettings))
	mux.HandleFunc("/api/settings/save", s.withAPIAuth(s.handleSaveSettings))

	// 静态文件 - 使用 web 子目录
	webFS, err := fs.Sub(webFiles, "web")
	if err != nil {
		return err
	}
	mux.Handle("/", http.FileServer(http.FS(webFS)))

	s.server = &http.Server{
		Addr:              listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	slog.Info("Web 服务器启动", "listen", listenAddr)

	// 启动服务器
	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("服务器错误", "error", err)
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		return err
	}

	// 优雅关闭
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// ModemInfo 调制解调器信息
type ModemInfo struct {
	IMEI          string `json:"imei"`
	Model         string `json:"model"`
	Manufacturer  string `json:"manufacturer"`
	Number        string `json:"number"`
	SignalQuality uint32 `json:"signal_quality"`
	OperatorName  string `json:"operator_name"`
	ICCID         string `json:"iccid"`
	State         string `json:"state"`
	DisplayName   string `json:"display_name"`
}

const maxRequestBodySize = 1 << 20 // 1MB

// handleModems 处理调制解调器信息请求
func (s *Server) handleModems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	modems := s.forwarder.GetModems()
	infos := make([]ModemInfo, 0, len(modems))

	s.cfgMu.RLock()
	cfg := s.cfg
	s.cfgMu.RUnlock()

	for _, modem := range modems {
		// 更新实时信息
		modem.UpdateSignalQuality()
		modem.UpdateOperatorName()
		modem.UpdateICCID()
		modem.UpdateState()

		info := ModemInfo{
			IMEI:          modem.EquipmentIdentifier,
			Model:         modem.Model,
			Manufacturer:  modem.Manufacturer,
			Number:        modem.Number,
			SignalQuality: modem.SignalQuality,
			OperatorName:  modem.OperatorName,
			ICCID:         modem.ICCID,
			State:         modem.State.String(),
			DisplayName:   cfg.ModemDisplayName(modem.EquipmentIdentifier),
		}

		infos = append(infos, info)
	}

	writeJSON(w, infos)
}

// handleMessages 处理短信列表请求
func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.storage == nil {
		writeJSON(w, map[string]interface{}{
			"items": []storage.Message{},
			"total": 0,
		})
		return
	}

	// 解析分页参数
	query := r.URL.Query()
	limit := 50
	offset := 0

	if l := query.Get("limit"); l != "" {
		v, err := strconv.Atoi(l)
		if err != nil {
			http.Error(w, "Invalid limit", http.StatusBadRequest)
			return
		}
		limit = v
	}
	if o := query.Get("offset"); o != "" {
		v, err := strconv.Atoi(o)
		if err != nil {
			http.Error(w, "Invalid offset", http.StatusBadRequest)
			return
		}
		offset = v
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	messages, total := s.storage.ListWithPagination(limit, offset)

	writeJSON(w, map[string]interface{}{
		"items": messages,
		"total": total,
	})
}

// handleUSSD 处理 USSD 请求
func (s *Server) handleUSSD(w http.ResponseWriter, r *http.Request) {
	// Helper to send JSON error
	sendError := func(w http.ResponseWriter, msg string, code int) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": msg,
		})
	}

	if r.Method != http.MethodPost {
		sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IMEI string `json:"imei"`
		Code string `json:"code"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// 校验 USSD 代码格式（通常以 * 或 # 开头，以 # 结尾，只包含数字、*、#）
	if req.Code == "" {
		sendError(w, "USSD code is empty", http.StatusBadRequest)
		return
	}
	for _, c := range req.Code {
		if c != '*' && c != '#' && (c < '0' || c > '9') {
			sendError(w, "Invalid USSD code format, only digits, * and # are allowed", http.StatusBadRequest)
			return
		}
	}

	modems := s.forwarder.GetModems()
	var targetModem *modem.Modem
	for _, m := range modems {
		if m.EquipmentIdentifier == req.IMEI {
			targetModem = m
			break
		}
	}

	if targetModem == nil {
		sendError(w, "Device not found", http.StatusNotFound)
		return
	}

	slog.Info("执行 USSD", "imei", req.IMEI, "code", req.Code)

	reply, err := targetModem.RunUSSD(req.Code)
	if err != nil {
		slog.Error("USSD 执行失败", "error", err)
		sendError(w, "执行失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{
		"reply": reply,
	})
}

// handleDeleteMessage 处理删除短信请求
func (s *Server) handleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.storage == nil {
		http.Error(w, "Storage not enabled", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		ID string `json:"id"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := s.storage.Delete(req.ID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, map[string]bool{"success": true})
}

// handleChannels 处理获取通道配置请求
func (s *Server) handleChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 定义所有预设通道模板（用于补齐用户未配置的通道类型）
	defaultChannels := map[string]config.ChannelConfig{
		"email": {
			Type:              "email",
			Enabled:           false,
			RequestTimeoutSec: 10,
			Port:              587,
			UseTLS:            true,
		},
		"bark": {
			Type:              "bark",
			Enabled:           false,
			RequestTimeoutSec: 10,
		},
		"gotify": {
			Type:              "gotify",
			Enabled:           false,
			RequestTimeoutSec: 10,
			Priority:          5,
		},
		"serverchan": {
			Type:              "serverchan",
			Enabled:           false,
			RequestTimeoutSec: 10,
		},
		"webhook": {
			Type:                "webhook",
			Enabled:             false,
			RequestTimeoutSec:   10,
			AllowPrivateNetwork: false,
			Method:              "POST",
		},
		"wecom": {
			Type:              "wecom",
			Enabled:           false,
			RequestTimeoutSec: 10,
		},
		"feishu": {
			Type:              "feishu",
			Enabled:           false,
			RequestTimeoutSec: 10,
			ReceiveIDType:     "open_id",
		},
		"dingtalk": {
			Type:              "dingtalk",
			Enabled:           false,
			RequestTimeoutSec: 10,
		},
		"telegram": {
			Type:              "telegram",
			Enabled:           false,
			RequestTimeoutSec: 10,
		},
	}

	// 以配置文件中的通道为基础，保留所有已配置的通道（含同类型多个）
	s.cfgMu.RLock()
	channels := make([]config.ChannelConfig, len(s.cfg.Channels))
	copy(channels, s.cfg.Channels)
	s.cfgMu.RUnlock()

	// 记录已配置的类型
	configuredTypes := make(map[string]bool)
	for _, ch := range channels {
		configuredTypes[ch.Type] = true
	}

	// 补齐未配置的类型
	for typeName, defaultCh := range defaultChannels {
		if !configuredTypes[typeName] {
			channels = append(channels, defaultCh)
		}
	}

	writeJSON(w, channels)
}

// handleSaveChannels 处理保存通道配置请求
func (s *Server) handleSaveChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var channels []config.ChannelConfig
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	if err := json.NewDecoder(r.Body).Decode(&channels); err != nil {
		http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 先热重载转发器中的通知器，验证配置有效性。
	if err := s.forwarder.ReloadChannels(channels); err != nil {
		http.Error(w, "Failed to reload channels: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 重载成功后再持久化到配置文件，避免写入无效配置。
	s.cfgMu.Lock()
	s.cfg.Channels = make([]config.ChannelConfig, len(channels))
	for i, ch := range channels {
		s.cfg.Channels[i] = config.CloneChannelConfig(ch)
	}
	saveErr := s.cfg.Save(s.configPath)
	s.cfgMu.Unlock()
	if saveErr != nil {
		http.Error(w, "Failed to save config: "+saveErr.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]bool{"success": true})
}

// handleTestChannel 处理测试通道请求
func (s *Server) handleTestChannel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var channels []config.ChannelConfig
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	if err := json.NewDecoder(r.Body).Decode(&channels); err != nil {
		http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}
	if !hasEnabledChannel(channels) {
		http.Error(w, "至少需要启用一个推送通道", http.StatusBadRequest)
		return
	}

	// 创建通知发送器（包含所有要测试的通道）
	testCfg := &config.Config{
		Channels: channels,
	}
	n, err := notifier.New(testCfg)
	if err != nil {
		http.Error(w, "创建通知器失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 创建测试消息
	s.cfgMu.RLock()
	testMsg := notifier.Message{
		Modem:             "测试设备",
		DeviceName:        s.cfg.DeviceName,
		ShowDeviceInTitle: s.cfg.DeviceNameInTitle,
		ShowDeviceInBody:  s.cfg.DeviceNameInBody,
		From:              "测试号码",
		Text:              "这是一条测试推送消息，如果您收到此消息，说明推送通道配置正确。",
		Timestamp:         time.Now(),
		Incoming:          true,
	}
	s.cfgMu.RUnlock()

	// 发送测试消息
	if err := n.Send(testMsg); err != nil {
		http.Error(w, "测试失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "测试消息已发送",
	})
}

func hasEnabledChannel(channels []config.ChannelConfig) bool {
	for _, ch := range channels {
		if ch.Enabled {
			return true
		}
	}
	return false
}

// handleWebSocket 处理 WebSocket 连接
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	authenticated, err := s.isAuthenticated(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.storage != nil {
		requiresSetup, err := s.requiresPasswordSetup()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if requiresSetup || !authenticated {
			state := authStatus{AuthEnabled: true, RequiresSetup: requiresSetup, Authenticated: false}
			code := http.StatusUnauthorized
			if requiresSetup {
				code = http.StatusPreconditionRequired
				state.Message = "当前实例尚未设置管理密码，请先完成初始化"
			} else {
				state.Message = "登录已失效，请重新登录"
			}
			writeAuthState(w, code, state)
			return
		}
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket 升级失败", "error", err)
		return
	}
	_ = conn.SetWriteDeadline(time.Time{})

	client := &wsClient{conn: conn}

	// 注册客户端
	s.clientsMu.Lock()
	s.clients[client] = true
	s.clientsMu.Unlock()

	slog.Info("WebSocket 客户端连接", "remote", r.RemoteAddr)

	// 客户端断开时清理
	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, client)
		s.clientsMu.Unlock()
		conn.Close()
		slog.Info("WebSocket 客户端断开", "remote", r.RemoteAddr)
	}()

	// 设置心跳检测
	const (
		pongWait   = 60 * time.Second
		pingPeriod = 50 * time.Second // 必须小于 pongWait
	)

	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// 启动 Ping 发送 goroutine
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				client.writeMu.Lock()
				err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second))
				client.writeMu.Unlock()
				if err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()
	defer close(done)

	// 保持连接并处理消息
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// BroadcastMessage 广播新消息给所有 WebSocket 客户端
func (s *Server) BroadcastMessage(msg storage.Message) {
	data, err := json.Marshal(map[string]interface{}{
		"type":    "new_message",
		"message": msg,
	})
	if err != nil {
		slog.Error("序列化消息失败", "error", err)
		return
	}

	s.clientsMu.RLock()
	clients := make([]*wsClient, 0, len(s.clients))
	for client := range s.clients {
		clients = append(clients, client)
	}
	s.clientsMu.RUnlock()

	for _, client := range clients {
		client.writeMu.Lock()
		_ = client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		err := client.conn.WriteMessage(websocket.TextMessage, data)
		client.writeMu.Unlock()
		if err != nil {
			slog.Error("发送 WebSocket 消息失败", "error", err)
			s.clientsMu.Lock()
			delete(s.clients, client)
			s.clientsMu.Unlock()
			_ = client.conn.Close()
		}
	}
}

// handleSettings 处理获取系统设置请求
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.cfgMu.RLock()
	data := map[string]any{
		"device_name":          s.cfg.DeviceName,
		"device_name_in_title": s.cfg.DeviceNameInTitle,
		"device_name_in_body":  s.cfg.DeviceNameInBody,
		"always_on_modems":     s.cfg.AlwaysOnModems,
		"modems":               s.cfg.Modems,
	}
	s.cfgMu.RUnlock()
	writeJSON(w, data)
}

// handleSaveSettings 处理保存系统设置请求
func (s *Server) handleSaveSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DeviceName        string               `json:"device_name"`
		DeviceNameInTitle bool                  `json:"device_name_in_title"`
		DeviceNameInBody  bool                  `json:"device_name_in_body"`
		AlwaysOnModems    bool                  `json:"always_on_modems"`
		Modems            []config.ModemConfig  `json:"modems"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	s.cfgMu.Lock()
	s.cfg.DeviceName = req.DeviceName
	s.cfg.DeviceNameInTitle = req.DeviceNameInTitle
	s.cfg.DeviceNameInBody = req.DeviceNameInBody
	s.cfg.AlwaysOnModems = req.AlwaysOnModems
	s.cfg.Modems = append([]config.ModemConfig(nil), req.Modems...)

	// 保存到配置文件
	if err := s.cfg.Save(s.configPath); err != nil {
		s.cfgMu.Unlock()
		http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.cfgMu.Unlock()
	s.forwarder.UpdateMessageTemplate(req.DeviceName, req.DeviceNameInTitle, req.DeviceNameInBody)
	s.forwarder.UpdateModemConfig(req.Modems, req.AlwaysOnModems)

	writeJSON(w, map[string]bool{"success": true})
}
