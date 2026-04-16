// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type GovGetVeLockLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGovGetVeLockLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GovGetVeLockLogic {
	return &GovGetVeLockLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GovGetVeLockLogic) GovGetVeLock() error {
	// todo: add your logic here and delete this line

	return nil
}
