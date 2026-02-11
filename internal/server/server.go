package server

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sms-forwarder/internal/config"
	"github.com/sms-forwarder/internal/forwarder"
	"github.com/sms-forwarder/internal/modem"
	"github.com/sms-forwarder/internal/notifier"
	"github.com/sms-forwarder/internal/storage"
)

//go:embed web/*
var webFiles embed.FS

// Server Web 服务器
type Server struct {
	cfg        *config.Config
	forwarder  *forwarder.Forwarder
	storage    *storage.Storage
	server     *http.Server
	configPath string
	clients    map[*websocket.Conn]bool
	clientsMu  sync.RWMutex
	upgrader   websocket.Upgrader
}

// New 创建服务器
func New(cfg *config.Config, fwd *forwarder.Forwarder, store *storage.Storage, configPath string) *Server {
	return &Server{
		cfg:        cfg,
		forwarder:  fwd,
		storage:    store,
		configPath: configPath,
		clients:    make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有源
			},
		},
	}
}

// Run 运行服务器
func (s *Server) Run(ctx context.Context) error {
	if !s.cfg.Server.Enabled {
		<-ctx.Done()
		return nil
	}

	mux := http.NewServeMux()

	// API 路由
	mux.HandleFunc("/api/modems", s.handleModems)
	mux.HandleFunc("/api/messages", s.handleMessages)
	mux.HandleFunc("/api/messages/delete", s.handleDeleteMessage)
	mux.HandleFunc("/api/channels", s.handleChannels)
	mux.HandleFunc("/api/channels/save", s.handleSaveChannels)
	mux.HandleFunc("/api/channels/test", s.handleTestChannel)
	mux.HandleFunc("/api/ussd", s.handleUSSD)
	mux.HandleFunc("/ws", s.handleWebSocket)

	// 静态文件 - 使用 web 子目录
	webFS, err := fs.Sub(webFiles, "web")
	if err != nil {
		return err
	}
	mux.Handle("/", http.FileServer(http.FS(webFS)))

	s.server = &http.Server{
		Addr:    s.cfg.Server.Listen,
		Handler: mux,
	}

	slog.Info("Web 服务器启动", "listen", s.cfg.Server.Listen)

	// 启动服务器
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("服务器错误", "error", err)
		}
	}()

	<-ctx.Done()

	// 优雅关闭
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.server.Shutdown(shutdownCtx)
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
}

// handleModems 处理调制解调器信息请求
func (s *Server) handleModems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	modems := s.forwarder.GetModems()
	infos := make([]ModemInfo, 0, len(modems))

	for _, modem := range modems {
		// 更新实时信息
		modem.UpdateSignalQuality()
		modem.UpdateOperatorName()
		modem.UpdateICCID()

		info := ModemInfo{
			IMEI:          modem.EquipmentIdentifier,
			Model:         modem.Model,
			Manufacturer:  modem.Manufacturer,
			Number:        modem.Number,
			SignalQuality: modem.SignalQuality,
			OperatorName:  modem.OperatorName,
			ICCID:         modem.ICCID,
		}

		infos = append(infos, info)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(infos)
}

// handleMessages 处理短信列表请求
func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if s.storage == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
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

	json.NewEncoder(w).Encode(map[string]interface{}{
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
		json.NewEncoder(w).Encode(map[string]string{
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

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request", http.StatusBadRequest)
		return
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
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

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := s.storage.Delete(req.ID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// handleChannels 处理获取通道配置请求
func (s *Server) handleChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 定义所有预设通道模板
	channels := []config.ChannelConfig{
		{
			Type:    "email",
			Enabled: false,
			Port:    587,
			UseTLS:  true,
		},
		{
			Type:    "bark",
			Enabled: false,
		},
		{
			Type:     "gotify",
			Enabled:  false,
			Priority: 5,
		},
		{
			Type:    "serverchan",
			Enabled: false,
		},
		{
			Type:    "webhook",
			Enabled: false,
			Method:  "POST",
		},
	}

	// 将配置文件中的值合并到模板中
	for i := range channels {
		for j := range s.cfg.Channels {
			if channels[i].Type == s.cfg.Channels[j].Type {
				channels[i] = s.cfg.Channels[j]
				break
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channels)
}

// handleSaveChannels 处理保存通道配置请求
func (s *Server) handleSaveChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var channels []config.ChannelConfig
	if err := json.NewDecoder(r.Body).Decode(&channels); err != nil {
		http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 先热重载转发器中的通知器
	if err := s.forwarder.ReloadChannels(channels); err != nil {
		http.Error(w, "Failed to reload channels: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 保存到配置文件
	if err := s.cfg.Save(s.configPath); err != nil {
		http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// handleTestChannel 处理测试通道请求
func (s *Server) handleTestChannel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var channels []config.ChannelConfig
	if err := json.NewDecoder(r.Body).Decode(&channels); err != nil {
		http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
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
	testMsg := notifier.Message{
		Modem:     "测试设备",
		From:      "测试号码",
		Text:      "这是一条测试推送消息，如果您收到此消息，说明推送通道配置正确。",
		Timestamp: time.Now(),
		Incoming:  true,
	}

	// 发送测试消息
	if err := n.Send(testMsg); err != nil {
		http.Error(w, "测试失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "测试消息已发送",
	})
}

// handleWebSocket 处理 WebSocket 连接
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket 升级失败", "error", err)
		return
	}

	// 注册客户端
	s.clientsMu.Lock()
	s.clients[conn] = true
	s.clientsMu.Unlock()

	slog.Info("WebSocket 客户端连接", "remote", r.RemoteAddr)

	// 客户端断开时清理
	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, conn)
		s.clientsMu.Unlock()
		conn.Close()
		slog.Info("WebSocket 客户端断开", "remote", r.RemoteAddr)
	}()

	// 保持连接并处理 ping/pong
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
	clients := make([]*websocket.Conn, 0, len(s.clients))
	for client := range s.clients {
		clients = append(clients, client)
	}
	s.clientsMu.RUnlock()

	for _, client := range clients {
		err := client.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			slog.Error("发送 WebSocket 消息失败", "error", err)
			s.clientsMu.Lock()
			delete(s.clients, client)
			s.clientsMu.Unlock()
			_ = client.Close()
		}
	}
}
