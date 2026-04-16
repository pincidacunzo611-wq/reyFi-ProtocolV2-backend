package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/reyfi/reyfi-backend/app/chain-indexer/internal/config"
	"github.com/reyfi/reyfi-backend/app/chain-indexer/internal/indexer"
	"github.com/reyfi/reyfi-backend/app/chain-indexer/internal/svc"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

var configFile = flag.String("f", "etc/indexer-dev.yaml", "config file path")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	ctx := svc.NewServiceContext(c)
	defer ctx.Close()

	// 创建并启动区块扫描器
	scanner := indexer.NewBlockScanner(ctx)

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logx.Info("chain indexer starting...")
		if err := scanner.Start(); err != nil {
			logx.Errorf("block scanner error: %v", err)
			quit <- syscall.SIGTERM
		}
	}()

	<-quit
	logx.Info("chain indexer shutting down...")
	scanner.Stop()
	logx.Info("chain indexer stopped")
}
