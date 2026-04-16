package main

import (
	"flag"
	"github.com/reyfi/reyfi-backend/app/governance/internal/config"
	"github.com/reyfi/reyfi-backend/app/governance/internal/server"
	"github.com/reyfi/reyfi-backend/app/governance/internal/svc"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/governance-dev.yaml", "config file path")

func main() {
	flag.Parse()
	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)
	go ctx.StartConsumers()
	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		server.RegisterGovernanceServiceServer(grpcServer, server.NewGovernanceServiceServer(ctx))
		if c.Mode == "dev" { reflection.Register(grpcServer) }
	})
	defer s.Stop()
	logx.Infof("governance rpc server starting at %s...", c.ListenOn)
	s.Start()
}
