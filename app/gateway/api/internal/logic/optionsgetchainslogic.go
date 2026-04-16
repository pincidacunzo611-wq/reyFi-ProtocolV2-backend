// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type OptionsGetChainsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewOptionsGetChainsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OptionsGetChainsLogic {
	return &OptionsGetChainsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *OptionsGetChainsLogic) OptionsGetChains() error {
	// todo: add your logic here and delete this line

	return nil
}
