// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/svc"
	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type LendingGetLiquidationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLendingGetLiquidationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LendingGetLiquidationsLogic {
	return &LendingGetLiquidationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LendingGetLiquidationsLogic) LendingGetLiquidations(req *types.LendingLiquidationsReq) error {
	// todo: add your logic here and delete this line

	return nil
}
