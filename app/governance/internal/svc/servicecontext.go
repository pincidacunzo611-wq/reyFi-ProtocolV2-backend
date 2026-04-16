package svc

import (
	"context"; "database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/reyfi/reyfi-backend/app/governance/internal/config"
	"github.com/reyfi/reyfi-backend/app/governance/internal/consumer"
	"github.com/reyfi/reyfi-backend/app/governance/internal/model"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config        config.Config; DB *sql.DB; Redis *redis.Redis
	ProposalModel *model.ProposalModel; VoteModel *model.VoteModel
	VeLockModel   *model.VeLockModel; GaugeModel *model.GaugeVoteModel
	govConsumer   *consumer.GovernanceEventConsumer
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := sql.Open("mysql", c.DataSource); if err != nil { logx.Must(err) }
	db.SetMaxOpenConns(50); db.SetMaxIdleConns(20)
	if err := db.Ping(); err != nil { logx.Must(err) }
	rds := redis.MustNewRedis(redis.RedisConf{Host: c.Redis.Host, Type: c.Redis.Type, Pass: c.Redis.Pass})
	s := &ServiceContext{Config: c, DB: db, Redis: rds,
		ProposalModel: model.NewProposalModel(db), VoteModel: model.NewVoteModel(db),
		VeLockModel: model.NewVeLockModel(db), GaugeModel: model.NewGaugeVoteModel(db),
	}
	s.govConsumer = consumer.NewGovernanceEventConsumer(db, c.Chain.ChainId, rds)
	return s
}

func (s *ServiceContext) StartConsumers() {
	cfg := s.Config.Kafka; if len(cfg.Brokers) == 0 { return }
	logx.Infof("starting governance kafka consumer: topic=%s", cfg.Topic)
	if err := s.govConsumer.Start(context.Background(), cfg.Brokers, cfg.GroupId, cfg.Topic); err != nil {
		logx.Errorf("governance consumer error: %v", err)
	}
}
