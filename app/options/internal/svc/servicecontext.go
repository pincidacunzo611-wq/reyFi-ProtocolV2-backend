package svc

import (
	"context"
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/reyfi/reyfi-backend/app/options/internal/config"
	"github.com/reyfi/reyfi-backend/app/options/internal/consumer"
	"github.com/reyfi/reyfi-backend/app/options/internal/model"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config          config.Config
	DB              *sql.DB
	Redis           *redis.Redis
	MarketModel     *model.OptionsMarketModel
	PositionModel   *model.OptionsPositionModel
	SettlementModel *model.OptionsSettlementModel
	VolModel        *model.OptionsVolSurfaceModel
	optionsConsumer *consumer.OptionsEventConsumer
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := sql.Open("mysql", c.DataSource)
	if err != nil { logx.Must(err) }
	db.SetMaxOpenConns(50); db.SetMaxIdleConns(20)
	if err := db.Ping(); err != nil { logx.Must(err) }

	rds := redis.MustNewRedis(redis.RedisConf{Host: c.Redis.Host, Type: c.Redis.Type, Pass: c.Redis.Pass})
	svcCtx := &ServiceContext{
		Config: c, DB: db, Redis: rds,
		MarketModel:     model.NewOptionsMarketModel(db),
		PositionModel:   model.NewOptionsPositionModel(db),
		SettlementModel: model.NewOptionsSettlementModel(db),
		VolModel:        model.NewOptionsVolSurfaceModel(db),
	}
	svcCtx.optionsConsumer = consumer.NewOptionsEventConsumer(db, c.Chain.ChainId, rds)
	return svcCtx
}

func (s *ServiceContext) StartConsumers() {
	cfg := s.Config.Kafka
	if len(cfg.Brokers) == 0 { return }
	logx.Infof("starting options kafka consumer: topic=%s", cfg.Topic)
	if err := s.optionsConsumer.Start(context.Background(), cfg.Brokers, cfg.GroupId, cfg.Topic); err != nil {
		logx.Errorf("options consumer error: %v", err)
	}
}
