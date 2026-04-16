// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/svc"
	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type LendingGetMarketDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLendingGetMarketDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LendingGetMarketDetailLogic {
	return &LendingGetMarketDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LendingGetMarketDetailLogic) LendingGetMarketDetail(req *types.LendingMarketDetailReq) error {
	// todo: add your logic here and delete this line

	return nil
}
