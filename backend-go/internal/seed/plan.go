package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
)

// plan.go — 写入计划（dry-run 输出 / 执行输入）与口径校验。
//
// BuildPlan 是纯计算入口：读取 //go:embed 素材、展开静态规格、计算 file_id/
// s2_cell_id/计数列，不连接数据库与对象存储。--dry-run 只调用它并打印计划；
// 真实执行则在计划之上再做清理/上传/写库。

// Mode 是 seed 的执行模式。
type Mode string

const (
	// ModeIfMissing 是默认模式：已存在的行跳过（保留开发中产生的互动）。
	ModeIfMissing Mode = "if-missing"
	// ModeReset 先按固定前缀删除 seed 数据（含 S3 对象）再重建。
	ModeReset Mode = "reset"
)

// Options 是 seed 的执行参数（来自 cmd/seed 的 flag 与 env）。
type Options struct {
	// Mode 为空时按 ModeIfMissing 处理。
	Mode Mode
	// DryRun 只打印计划，不触达 DB / 对象存储。
	DryRun bool
	// Now 是数据时间线的基准时刻（零值取 time.Now()）；内部向下取整到小时，
	// 保证同一小时内重复 --reset 得到相同时间线。
	Now time.Time
	// QAPassword / AdminPassword 分别覆盖两个 QA 账号的口令；
	// 空值回退规格缺省值（仅本地开发使用）。
	QAPassword    string
	AdminPassword string
	// BcryptCost 是账号口令哈希成本；0 取 auth.DefaultBcryptCost()。
	BcryptCost int
}

// normalize 补齐缺省值并规范化 Mode/时间。
func (o Options) normalize() (Options, error) {
	normalized := o
	switch normalized.Mode {
	case "":
		normalized.Mode = ModeIfMissing
	case ModeIfMissing, ModeReset:
	default:
		return Options{}, fmt.Errorf("未知执行模式 %q（仅支持 %s / %s）", normalized.Mode, ModeIfMissing, ModeReset)
	}
	if normalized.Now.IsZero() {
		normalized.Now = time.Now().UTC()
	}
	normalized.Now = normalized.Now.UTC().Truncate(time.Hour)
	if normalized.QAPassword == "" {
		normalized.QAPassword = defaultQAPassword
	}
	if normalized.AdminPassword == "" {
		normalized.AdminPassword = defaultAdminPassword
	}
	if normalized.BcryptCost == 0 {
		// 与 auth 模块的口令哈希成本同源：modules/auth 启动时把该常量写入
		// 全局 hashCost（auth.DefaultBcryptCost = config.DefaultBcryptCost）。
		normalized.BcryptCost = config.DefaultBcryptCost()
	}
	return normalized, nil
}

// Plan 是一次 seed 的完整写入计划。
type Plan struct {
	Options  Options
	BaseTime time.Time
	accounts []accountSpec
	steps    []tableData
	media    []mediaObject
	assets   map[string]seedAsset
}

// TableCount 是一张表的写入行数（dry-run 摘要）。
type TableCount struct {
	Table string
	Rows  int
}

// TableCounts 按首次出现顺序聚合各表行数（comments 分两批写入会合并计数）。
func (p *Plan) TableCounts() []TableCount {
	order := make([]string, 0, len(p.steps))
	counts := make(map[string]int, len(p.steps))
	for _, step := range p.steps {
		if _, ok := counts[step.Table]; !ok {
			order = append(order, step.Table)
		}
		counts[step.Table] += len(step.Rows)
	}
	out := make([]TableCount, 0, len(order))
	for _, table := range order {
		out = append(out, TableCount{Table: table, Rows: counts[table]})
	}
	return out
}

// RowCount 是全部表的数据总行数。
func (p *Plan) RowCount() int {
	return countRows(p.steps)
}

// MediaCount 是待上传对象总数。
func (p *Plan) MediaCount() int {
	return len(p.media)
}

// Accounts 返回账号规格（dry-run 展示用；不包含口令）。
func (p *Plan) Accounts() []accountSpec {
	return append([]accountSpec(nil), p.accounts...)
}

// BuildPlan 构建写入计划。素材读取/规格展开都在这里完成；失败说明素材缺失、
// 规格引用了不存在的实体或坐标非法。
func BuildPlan(cfg config.Config, opts Options) (*Plan, error) {
	normalized, err := opts.normalize()
	if err != nil {
		return nil, err
	}
	data := buildSeedData()
	assets, err := loadAllAssets()
	if err != nil {
		return nil, err
	}
	mediaBundle, err := buildMediaPlan(normalized.Now, data, assets)
	if err != nil {
		return nil, err
	}
	appConfigJSON, err := buildAppConfigJSON(cfg, normalized.Now)
	if err != nil {
		return nil, err
	}
	dataset, err := buildDataset(data, mediaBundle, buildDatasetOptions{
		baseTime:      normalized.Now,
		appConfigJSON: appConfigJSON,
		qaPassword:    normalized.QAPassword,
		adminPassword: normalized.AdminPassword,
	})
	if err != nil {
		return nil, err
	}
	return &Plan{
		Options:  normalized,
		BaseTime: normalized.Now,
		accounts: dataset.accounts,
		steps:    dataset.steps,
		media:    mediaBundle.objects,
		assets:   assets,
	}, nil
}

// buildAppConfigJSON 生成 app_configs.payload_json（config_id='default'）。
//
// payload 是完整的 AppConfig 结构（platform/appconfig 的持久化格式）：以
// NewStaticSource 的默认配置为底，写入 seed 的更新时间/操作者。不写这行时服务
// 会用默认配置（行为等价），写这行的目的是让管理端「应用配置」页有真实数据。
func buildAppConfigJSON(cfg config.Config, baseTime time.Time) (string, error) {
	appConfig := appconfig.NewStaticSource(cfg).Get(context.Background())
	appConfig.UpdatedAt = formatSeedTime(baseTime)
	appConfig.UpdatedByUserID = seedAppConfigUpdatedBy
	// 默认浏览锚点对齐种子数据所在地（见 specs.go 的 seedAnchor*）：服务端默认值来自
	// 运行时 config（南天门），不改会与徐州种子数据相距 400+km，附近/地图页开箱即空。
	seedAnchor := appconfig.Location{
		Point:        appconfig.GeoPoint{Latitude: seedAnchorLatitude, Longitude: seedAnchorLongitude},
		LocationName: seedAnchorLocationName,
	}
	appConfig.Defaults.DefaultLocation = seedAnchor
	appConfig.Defaults.DefaultGroupSettings.Location = seedAnchor
	encoded, err := json.Marshal(appConfig)
	if err != nil {
		return "", fmt.Errorf("序列化应用配置失败: %w", err)
	}
	return string(encoded), nil
}

// PrintPlan 输出版本化的 dry-run 计划（不含任何口令明文）。
func PrintPlan(w io.Writer, plan *Plan) error {
	if _, err := fmt.Fprintf(w, "HotSpot 开发种子数据计划\n"); err != nil {
		return err
	}
	fmt.Fprintf(w, "  模式        : %s\n", plan.Options.Mode)
	fmt.Fprintf(w, "  基准时间    : %s（UTC，向下取整到小时）\n", plan.BaseTime.Format(time.RFC3339))
	fmt.Fprintf(w, "  媒体对象    : %d 个（%s）\n", plan.MediaCount(), formatCategoryCounts(plan.media))
	fmt.Fprintf(w, "  数据总行数  : %d\n", plan.RowCount())
	fmt.Fprintln(w, "\n账号（口令来自 SEED_QA_PASSWORD / SEED_ADMIN_PASSWORD，缺省为 dev 规格值）:")
	for _, account := range plan.accounts {
		role := "普通用户"
		switch account.UserID {
		case "user_seed_lumos":
			role = "QA 主账号（普通用户）"
		case "user_seed_admin":
			role = "管理员（grant_all，同时是完整普通用户）"
		}
		fmt.Fprintf(w, "  %-22s → %-20s %s\n", account.LoginID, account.UserID, role)
	}
	fmt.Fprintln(w, "\n数据表（按写入顺序）:")
	counts := plan.TableCounts()
	totalTables := 0
	for _, count := range counts {
		if count.Rows == 0 {
			continue
		}
		totalTables++
		fmt.Fprintf(w, "  %-28s %5d 行\n", count.Table, count.Rows)
	}
	fmt.Fprintf(w, "  %-28s %5d 张表\n", "合计", totalTables)
	if plan.Options.Mode == ModeReset {
		fmt.Fprintln(w, "\n--reset：执行时先按固定前缀删除 seed 数据（含对象存储对象），再重建。")
	} else {
		fmt.Fprintln(w, "\n--if-missing：执行时已存在的行跳过（保留开发中产生的互动）。")
	}
	fmt.Fprintln(w, "\n这是 --dry-run，未写入任何数据。去掉 --dry-run 即可实际执行。")
	return nil
}

// formatCategoryCounts 汇总媒体分类（按分类名排序，输出稳定）。
func formatCategoryCounts(media []mediaObject) string {
	counts := make(map[string]int, 4)
	for _, obj := range media {
		counts[obj.Category]++
	}
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	out := ""
	for i, name := range names {
		if i > 0 {
			out += " / "
		}
		out += fmt.Sprintf("%s %d", name, counts[name])
	}
	return out
}
