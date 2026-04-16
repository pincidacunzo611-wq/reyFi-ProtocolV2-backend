package svc

import (
	"context"; "database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/reyfi/reyfi-backend/app/vault/internal/config"
	"github.com/reyfi/reyfi-backend/app/vault/internal/consumer"
	"github.com/reyfi/reyfi-backend/app/vault/internal/model"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config        config.Config; DB *sql.DB; Redis *redis.Redis
	VaultModel    *model.VaultModel; SnapshotModel *model.VaultSnapshotModel
	PositionModel *model.VaultUserPositionModel; HarvestModel *model.VaultHarvestModel
	vaultConsumer *consumer.VaultEventConsumer
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := sql.Open("mysql", c.DataSource); if err != nil { logx.Must(err) }
	db.SetMaxOpenConns(50); db.SetMaxIdleConns(20)
	if err := db.Ping(); err != nil { logx.Must(err) }
	rds := redis.MustNewRedis(redis.RedisConf{Host: c.Redis.Host, Type: c.Redis.Type, Pass: c.Redis.Pass})
	s := &ServiceContext{Config: c, DB: db, Redis: rds,
		VaultModel: model.NewVaultModel(db), SnapshotModel: model.NewVaultSnapshotModel(db),
		PositionModel: model.NewVaultUserPositionModel(db), HarvestModel: model.NewVaultHarvestModel(db),
	}
	s.vaultConsumer = consumer.NewVaultEventConsumer(db, c.Chain.ChainId, rds)
	return s
}

func (s *ServiceContext) StartConsumers() {
	cfg := s.Config.Kafka; if len(cfg.Brokers) == 0 { return }
	logx.Infof("starting vault kafka consumer: topic=%s", cfg.Topic)
	if err := s.vaultConsumer.Start(context.Background(), cfg.Brokers, cfg.GroupId, cfg.Topic); err != nil {
		logx.Errorf("vault consumer error: %v", err)
	}
}
