// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/svc"
	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type FuturesBuildAdjustMarginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewFuturesBuildAdjustMarginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FuturesBuildAdjustMarginLogic {
	return &FuturesBuildAdjustMarginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FuturesBuildAdjustMarginLogic) FuturesBuildAdjustMargin(req *types.FuturesBuildAdjustMarginReq) error {
	// todo: add your logic here and delete this line

	return nil
}
