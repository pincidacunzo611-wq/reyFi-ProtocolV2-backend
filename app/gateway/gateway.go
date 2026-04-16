package main

import (
	"flag"
	"net/http"

	"github.com/reyfi/reyfi-backend/app/gateway/internal/config"
	"github.com/reyfi/reyfi-backend/app/gateway/internal/handler"
	"github.com/reyfi/reyfi-backend/app/gateway/internal/svc"
	"github.com/reyfi/reyfi-backend/pkg/response"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/gateway-dev.yaml", "config file path")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf,
		rest.WithCors(), // 开启 CORS
	)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)

	// 注册路由
	handler.RegisterHandlers(server, ctx)

	// 自定义错误处理
	httpx_errorHandler := func(err error) (int, interface{}) {
		return http.StatusOK, &response.Body{
			Code:    9001,
			Message: err.Error(),
		}
	}
	_ = httpx_errorHandler // 预留

	logx.Infof("gateway starting at %s:%d...", c.Host, c.Port)
	server.Start()
}
