package main

import (
	"flag"
	"github.com/reyfi/reyfi-backend/app/bot/internal/config"
	"github.com/reyfi/reyfi-backend/app/bot/internal/jobs"
	"github.com/reyfi/reyfi-backend/app/bot/internal/svc"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

var configFile = flag.String("f", "etc/bot-dev.yaml", "config file path")

func main() {
	flag.Parse()
	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	logx.Infof("bot service starting...")

	// 启动所有自动化任务
	scheduler := jobs.NewScheduler(ctx)
	scheduler.Start()

	// 阻塞等待信号
	select {}
}
