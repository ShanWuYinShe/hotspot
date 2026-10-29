package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/seed"
)

// cmd/seed — 开发种子数据命令的薄入口：
// 解析 flag → config.Load → persistence.NewFactory → seed.Run。
//
// 正式入口为仓库根 ./scripts/seed-dev.sh（在 compose 的 backend 容器内执行，
// 自动继承 POSTGRES_DSN / S3_* 等环境变量）。直接 go run 时需自备这些环境变量。
//
// 素材是外置的开发数据配套设施（仓库根 devdata/seed/media/），不打包进后端
// 二进制：宿主机默认解析 ../devdata/seed/media，dev 容器内由 dev compose 把
// devdata/ 挂载到 /devdata，脚本再显式传 -assets-dir 兜底。说明见 devdata/seed/README.md。
func main() {
	var (
		reset     = flag.Bool("reset", false, "先按固定前缀删除 seed 数据（含 S3 对象）再重建")
		dryRun    = flag.Bool("dry-run", false, "只打印写入计划，不写数据库/对象存储")
		assetsDir = flag.String("assets-dir", "", "开发素材目录（默认 ../devdata/seed/media 等候选；env SEED_ASSETS_DIR 等价）")
	)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `HotSpot 开发种子数据命令（仅限本地开发环境）

用法：
  go run ./cmd/seed [--reset] [--dry-run] [-assets-dir DIR]

模式：
  默认（--if-missing）  缺失则写入；已存在的行跳过，保留开发中产生的互动
  --reset               先按固定前缀清理 seed 数据（含 S3 对象），再完整重建
  --dry-run             只打印计划（素材目录/账号/媒体数量/各表行数），不写任何数据

素材（开发数据配套设施，外置、不打包进二进制）：
  默认依次尝试 ../devdata/seed/media、/devdata/seed/media、/src/devdata/seed/media
  （backend-go 目录下运行命中仓库根 devdata/seed/media；dev 容器内 devdata/ 挂载在
  /devdata）；也可用 -assets-dir 或 env SEED_ASSETS_DIR 显式指定。缺少素材会直接
  报错并列出缺失文件，不会静默跳过。说明见 devdata/seed/README.md。

账号（本地开发固定值，口令可用 env 覆盖）：
  qa@hotspot.local     QA 主账号（普通用户）   口令 env SEED_QA_PASSWORD（默认 Qa12345678）
  admin@hotspot.local  管理员（grant_all）      口令 env SEED_ADMIN_PASSWORD（默认 Admin12345678）

环境变量（容器内由 compose 注入）：
  POSTGRES_DSN  数据库连接串（必填，非 dry-run）
  S3_ENDPOINT / S3_BUCKET / S3_ACCESS_KEY / S3_SECRET_KEY  对象存储（必填，非 dry-run）
  SEED_QA_PASSWORD / SEED_ADMIN_PASSWORD                   可选，覆盖 QA 账号口令
  SEED_ASSETS_DIR                                          可选，等价于 -assets-dir

示例（仓库根执行，走 dev compose 容器）：
  ./scripts/seed-dev.sh            # 幂等注入
  ./scripts/seed-dev.sh --reset    # 清空后重建
  ./scripts/seed-dev.sh --dry-run  # 预览计划

宿主机直接 dry-run（不需要 DB，用于验证素材加载与计划）：
  cd backend-go && go run ./cmd/seed --dry-run -assets-dir ../devdata/seed/media

数据规格与实现：backend-go/internal/seed/；开发数据配套设施：devdata/seed/（README.md）。
`)
	}
	flag.Parse()
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "未知参数: %v（本命令不接受位置参数）\n\n", flag.Args())
		flag.Usage()
		os.Exit(2)
	}

	opts := seed.Options{
		Mode:          seed.ModeIfMissing,
		DryRun:        *dryRun,
		AssetsDir:     *assetsDir,
		QAPassword:    os.Getenv("SEED_QA_PASSWORD"),
		AdminPassword: os.Getenv("SEED_ADMIN_PASSWORD"),
	}
	if opts.AssetsDir == "" {
		// -assets-dir 优先；未显式传时允许用 env 指定（如 docker exec -e SEED_ASSETS_DIR=...）。
		opts.AssetsDir = os.Getenv("SEED_ASSETS_DIR")
	}
	if *reset {
		opts.Mode = seed.ModeReset
	}

	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// --dry-run 不需要数据库连接：不创建连接池，避免在未配置 DSN 的机器上误报。
	if opts.DryRun {
		if err := seed.Run(ctx, cfg, nil, opts); err != nil {
			log.Fatalf("seed 失败: %v", err)
		}
		return
	}

	factory := persistence.NewFactory(cfg)
	if factory.DB != nil {
		defer func() {
			if err := factory.DB.Close(); err != nil {
				log.Printf("关闭数据库连接失败: %v", err)
			}
		}()
	}
	if err := seed.Run(ctx, cfg, factory, opts); err != nil {
		log.Fatalf("seed 失败: %v", err)
	}
}
