package main

import (
	"flag"

	"github.com/reyfi/reyfi-backend/app/dex/internal/config"
	"github.com/reyfi/reyfi-backend/app/dex/internal/server"
	"github.com/reyfi/reyfi-backend/app/dex/internal/svc"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/dex-dev.yaml", "config file path")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	ctx := svc.NewServiceContext(c)

	// 启动 Kafka 消费者
	go ctx.StartConsumers()

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		server.RegisterDexServiceServer(grpcServer, server.NewDexServiceServer(ctx))

		// 开发模式开启 gRPC reflection
		if c.Mode == "dev" {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	logx.Infof("dex rpc server starting at %s...", c.ListenOn)
	s.Start()
}
