// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/svc"
	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type VaultGetDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewVaultGetDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VaultGetDetailLogic {
	return &VaultGetDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *VaultGetDetailLogic) VaultGetDetail(req *types.VaultDetailReq) error {
	// todo: add your logic here and delete this line

	return nil
}
