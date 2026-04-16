// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type FuturesGetPositionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewFuturesGetPositionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FuturesGetPositionsLogic {
	return &FuturesGetPositionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FuturesGetPositionsLogic) FuturesGetPositions() error {
	// todo: add your logic here and delete this line

	return nil
}
