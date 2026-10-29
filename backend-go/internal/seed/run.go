package seed

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"hotspot/backend-go/internal/modules/media"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// run.go — seed 执行入口：清理（可选）→ 上传对象 → 写库，并输出摘要。
//
// 依赖边界（可测试性）：
//   - 数据库通过 persistence.RunInTransaction + persistence.Queryable 使用，
//     不在本包内持有 *pgxpool；
//   - 对象存储通过 ObjectStorage 最小接口使用（media.Provider 的超集），
//     单测可注入 fake 记录 PUT/DELETE；
//   - 所有输出走 log（与仓库其它命令行工具一致），不引入 observability。

// ObjectStorage 是 seed 需要的最小对象存储接口：
// media.Provider 实现了它（PutObject / Delete）。
type ObjectStorage interface {
	PutObject(ctx context.Context, storageKey, contentType string, data []byte) error
	Delete(ctx context.Context, storageKey string) error
}

// txRunner 抽象事务执行（生产为 persistence.RunInTransaction；单测注入
// persistence.WithTx 包装的 fake 连接）。
type txRunner func(ctx context.Context, fn func(ctx context.Context) error) error

// Run 依据 Options 执行 seed：dry-run 只打印计划；否则先校验环境/依赖，
// 再按模式清理并双写 Postgres + 对象存储。
//
// 安全约束：只允许 dev/test 环境执行（APP_ENV=production 等直接拒绝），
// 避免把固定口令的开发账号写进生产库。
func Run(ctx context.Context, cfg config.Config, factory *persistence.Factory, opts Options) error {
	plan, err := BuildPlan(cfg, opts)
	if err != nil {
		return err
	}
	if plan.Options.DryRun {
		return PrintPlan(os.Stdout, plan)
	}
	if cfg.IsProductionLike() {
		return fmt.Errorf("开发种子数据只用于 dev/test 环境（当前 APP_ENV=%q）：请设置 APP_ENV=dev 后重试", cfg.AppEnv)
	}
	if factory == nil || factory.DB == nil {
		return errors.New("未配置数据库连接：请设置 POSTGRES_DSN（容器内由 compose 注入）")
	}
	if _, ok := factory.DB.(persistence.QueryableDB); !ok {
		return errors.New("数据库连接不可用于执行 SQL（persistence.QueryableDB 断言失败）：请检查 POSTGRES_DSN")
	}
	if cfg.S3Endpoint == "" || cfg.S3Bucket == "" || cfg.S3AccessKey == "" {
		return errors.New("对象存储未配置：需要 S3_ENDPOINT / S3_BUCKET / S3_ACCESS_KEY / S3_SECRET_KEY（容器内由 compose 注入）")
	}
	provider := media.NewObjectStorageProvider(cfg)
	runInTx := func(ctx context.Context, fn func(ctx context.Context) error) error {
		return persistence.RunInTransaction(ctx, factory, fn)
	}
	return execute(ctx, runInTx, provider, plan)
}

// execute 是 Run 的依赖注入版本（单测直接调用，注入 fake 事务/对象存储）。
func execute(ctx context.Context, runInTx txRunner, storage ObjectStorage, plan *Plan) error {
	startedAt := time.Now()
	log.Printf("seed: 开始执行（模式=%s，基准时间=%s）", plan.Options.Mode, plan.BaseTime.Format(time.RFC3339))

	if plan.Options.Mode == ModeReset {
		log.Printf("seed: 按固定前缀清理已有 seed 数据 ...")
		var existingKeys []string
		err := runInTx(ctx, func(txCtx context.Context) error {
			queryable, ok := persistence.QueryableFromContext(txCtx)
			if !ok {
				return errors.New("无法获取事务连接")
			}
			keys, err := cleanup(txCtx, queryable)
			if err != nil {
				return err
			}
			existingKeys = keys
			return nil
		})
		if err != nil {
			return fmt.Errorf("清理 seed 数据失败: %w", err)
		}
		keysToDelete := mergeStorageKeys(existingKeys, planStorageKeys(plan))
		log.Printf("seed: 删除 %d 个历史媒体对象 ...", len(keysToDelete))
		if err := deleteObjects(ctx, storage, keysToDelete); err != nil {
			return err
		}
	}

	log.Printf("seed: 上传 %d 个媒体对象到对象存储 ...", plan.MediaCount())
	if err := uploadMedia(ctx, storage, plan); err != nil {
		return err
	}

	log.Printf("seed: 写入 Postgres（%d 张表 / %d 行）...", len(plan.TableCounts()), plan.RowCount())
	err := runInTx(ctx, func(txCtx context.Context) error {
		queryable, ok := persistence.QueryableFromContext(txCtx)
		if !ok {
			return errors.New("无法获取事务连接")
		}
		return insertAll(txCtx, queryable, plan.steps, plan.Options.BcryptCost)
	})
	if err != nil {
		return fmt.Errorf("写入 seed 数据失败: %w", err)
	}

	log.Printf("seed: 完成（耗时 %s，模式=%s，%d 行数据 / %d 个媒体对象）",
		time.Since(startedAt).Round(time.Millisecond), plan.Options.Mode, plan.RowCount(), plan.MediaCount())
	printAccountHints()
	return nil
}

// printAccountHints 输出登录提示（不含口令明文；口令来源见 AGENTS/README）。
func printAccountHints() {
	log.Printf("seed: QA 账号 qa@hotspot.local（普通用户）/ admin@hotspot.local（管理员，grant_all）")
	log.Printf("seed: 口令来自 SEED_QA_PASSWORD / SEED_ADMIN_PASSWORD，缺省值见 backend-go/AGENTS.md「开发种子数据」")
}

// uploadMedia 把计划的全部媒体对象 PUT 到对象存储（同 key 覆盖，天然幂等）。
func uploadMedia(ctx context.Context, storage ObjectStorage, plan *Plan) error {
	for index, obj := range plan.media {
		asset, ok := plan.assets[obj.AssetName]
		if !ok {
			return fmt.Errorf("上传媒体 %s 失败：素材 %s 未加载", obj.StorageKey, obj.AssetName)
		}
		if err := storage.PutObject(ctx, obj.StorageKey, obj.MimeType, asset.Data); err != nil {
			return fmt.Errorf("上传媒体 %s 失败: %w", obj.StorageKey, err)
		}
		log.Printf("  [%d/%d] PUT %s", index+1, len(plan.media), obj.StorageKey)
	}
	return nil
}

// deleteObjects 删除对象存储中的历史 seed 对象（对象不存在视为已删除）。
func deleteObjects(ctx context.Context, storage ObjectStorage, keys []string) error {
	for _, key := range keys {
		if err := storage.Delete(ctx, key); err != nil {
			return fmt.Errorf("删除历史媒体对象 %s 失败: %w", key, err)
		}
	}
	return nil
}

// planStorageKeys 返回计划内全部对象的 storage_key。
func planStorageKeys(plan *Plan) []string {
	keys := make([]string, 0, len(plan.media))
	for _, obj := range plan.media {
		keys = append(keys, obj.StorageKey)
	}
	return keys
}

// mergeStorageKeys 合并历史与计划对象键（去重并排序，保证日志稳定）。
func mergeStorageKeys(keysA, keysB []string) []string {
	seen := make(map[string]struct{}, len(keysA)+len(keysB))
	merged := make([]string, 0, len(keysA)+len(keysB))
	for _, keys := range [][]string{keysA, keysB} {
		for _, key := range keys {
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			merged = append(merged, key)
		}
	}
	sort.Strings(merged)
	return merged
}
