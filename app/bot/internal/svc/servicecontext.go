package svc

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/reyfi/reyfi-backend/app/bot/internal/config"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config config.Config
	DB     *sql.DB
	Redis  *redis.Redis
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := sql.Open("mysql", c.DataSource)
	if err != nil { logx.Must(err) }
	db.SetMaxOpenConns(30); db.SetMaxIdleConns(10)
	if err := db.Ping(); err != nil { logx.Must(err) }
	logx.Info("bot database connected")

	rds := redis.MustNewRedis(redis.RedisConf{Host: c.Redis.Host, Type: c.Redis.Type, Pass: c.Redis.Pass})
	return &ServiceContext{Config: c, DB: db, Redis: rds}
}
