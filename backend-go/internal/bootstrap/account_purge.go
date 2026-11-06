package bootstrap

import (
	"context"
	"fmt"
)

// accountPurgeTarget 是账户注销级联参与者：各模块 Service 实现的
// PurgeUserData 只删除本模块归属的该用户数据，幂等。
type accountPurgeTarget interface {
	PurgeUserData(ctx context.Context, userID string) error
}

// accountPurgeCoordinator 领域归属清理的应用级编排（组合根策略）：按固定顺序
// 调用各模块清理，任一步失败即中止整个注销（已清模块幂等，下一次重试可继续）。
type accountPurgeCoordinator struct {
	targets []accountPurgeTarget
}

func (c accountPurgeCoordinator) PurgeLinkedData(ctx context.Context, userID string) error {
	for _, target := range c.targets {
		if err := target.PurgeUserData(ctx, userID); err != nil {
			return fmt.Errorf("account purge: %w", err)
		}
	}
	return nil
}
