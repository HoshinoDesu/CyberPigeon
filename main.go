package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/HoshinoDesu/CyberPigeon/internal/config"
	"github.com/HoshinoDesu/CyberPigeon/internal/forwarder"
	"github.com/HoshinoDesu/CyberPigeon/internal/modem"
	"github.com/HoshinoDesu/CyberPigeon/internal/server"
	"github.com/HoshinoDesu/CyberPigeon/internal/storage"
	"golang.org/x/term"
)

func main() {
	configFile := flag.String("config", "config.toml", "配置文件路径")
	resetPassword := flag.String("reset-password", "", "重置 Web 管理密码并退出；传入 - 表示从终端安全输入")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化存储
	var store *storage.Storage
	if cfg.Storage.Enabled {
		store, err = storage.New(cfg.Storage.Path)
		if err != nil {
			log.Fatalf("初始化存储失败: %v", err)
		}
		defer store.Close()
	}

	if *resetPassword != "" {
		if store == nil {
			log.Fatalf("无法重置密码：当前未启用存储")
		}
		password, err := resolveResetPassword(*resetPassword)
		if err != nil {
			log.Fatalf("读取重置密码失败: %v", err)
		}
		if err := store.SetPassword(password); err != nil {
			log.Fatalf("重置密码失败: %v", err)
		}
		if err := store.DeleteAllSessions(); err != nil {
			log.Fatalf("清理旧会话失败: %v", err)
		}
		fmt.Println("密码已重置成功，所有已登录会话已失效。")
		return
	}

	// 初始化调制解调器管理器
	manager, err := modem.NewManager()
	if err != nil {
		log.Fatalf("初始化 ModemManager 失败: %v", err)
	}
	defer manager.Close()

	// 初始化转发器
	fwd, err := forwarder.New(cfg, manager, store)
	if err != nil {
		log.Fatalf("初始化转发器失败: %v", err)
	}

	// 初始化 Web 服务器
	srv := server.New(cfg, fwd, store, *configFile)

	// 设置新消息处理器
	if store != nil {
		store.SetMessageHandler(func(msg storage.Message) {
			srv.BroadcastMessage(msg)
		})
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// 启动转发器
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := fwd.Run(ctx); err != nil {
			slog.Error("转发器运行错误", "error", err)
		}
	}()

	// 启动 Web 服务器
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.Run(ctx); err != nil {
			slog.Error("服务器运行错误", "error", err)
		}
	}()

	slog.Info("CyberPigeon 已启动")

	// 等待信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	slog.Info("收到停止信号，正在关闭...")
	cancel()
	wg.Wait() // 等待所有协程优雅退出
}

func resolveResetPassword(value string) (string, error) {
	if strings.TrimSpace(value) != "-" {
		return value, nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		data, err := io.ReadAll(io.LimitReader(os.Stdin, 4096))
		if err != nil {
			return "", err
		}
		password := strings.TrimSpace(string(data))
		if password == "" {
			return "", fmt.Errorf("未从标准输入读取到密码")
		}
		return password, nil
	}

	fmt.Print("请输入新密码: ")
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	password := strings.TrimSpace(string(passwordBytes))
	if password == "" {
		return "", fmt.Errorf("密码不能为空")
	}
	return password, nil
}
