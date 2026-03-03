package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/CyberPigeon/internal/config"
	"github.com/CyberPigeon/internal/forwarder"
	"github.com/CyberPigeon/internal/modem"
	"github.com/CyberPigeon/internal/server"
	"github.com/CyberPigeon/internal/storage"
)

func main() {
	configFile := flag.String("config", "config.toml", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化调制解调器管理器
	manager, err := modem.NewManager()
	if err != nil {
		log.Fatalf("初始化 ModemManager 失败: %v", err)
	}
	defer manager.Close()

	// 初始化存储
	var store *storage.Storage
	if cfg.Storage.Enabled {
		store, err = storage.New(cfg.Storage.Path)
		if err != nil {
			log.Fatalf("初始化存储失败: %v", err)
		}
		defer store.Close()
	}

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
