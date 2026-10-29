package seed

import (
	"context"
	"fmt"
	"sort"

	"golang.org/x/crypto/bcrypt"

	"hotspot/backend-go/internal/platform/persistence"
)

// tables.go — 表写入层：表/列描述 + 参数化 INSERT + 口令哈希占位替换。
//
// 幂等策略（seed-spec §1）：所有 INSERT 都是
//   INSERT ... ON CONFLICT DO NOTHING
// 因此默认 --if-missing 语义天然成立（已存在的行跳过、开发中产生的互动保留）；
// --reset 模式下先按固定前缀删除再调用同一写入逻辑，结果等价于重建。
//
// 只用 ON CONFLICT（不指定目标）而不是 ON CONFLICT (pk)：部分表还存在 PK 之外
// 的唯一约束（shares.share_code、admin_roles.name、reports 的
// (creator,target_type,target_id)、group_applications 的 pending 部分唯一索引），
// 无目标形式能兜住任意唯一冲突，避免 seed 与手工数据撞约束时整体失败。

// column 描述一列及其绑定参数的 SQL 转型（如 ::jsonb）。
// 绝大多数列由 pgx 按 Go 类型推导，无需转型；jsonb 列显式转型是为了让
// string 参数按 jsonb 而非 text 传入。
type column struct {
	name string
	cast string
}

func col(name string) column      { return column{name: name} }
func jsonbCol(name string) column { return column{name: name, cast: "::jsonb"} }

// tableData 是一张表的待写入行。
type tableData struct {
	Table   string
	Columns []column
	Rows    [][]any
}

// addRow 追加一行（参数顺序必须与 Columns 一致；arity 由 insertAll 校验）。
func (t *tableData) addRow(values ...any) {
	t.Rows = append(t.Rows, values)
}

// insertSQL 生成带占位符的 INSERT 语句。
func (t tableData) insertSQL() string {
	names := make([]string, 0, len(t.Columns))
	placeholders := make([]string, 0, len(t.Columns))
	for i, column := range t.Columns {
		names = append(names, column.name)
		placeholders = append(placeholders, fmt.Sprintf("$%d%s", i+1, column.cast))
	}
	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT DO NOTHING",
		t.Table,
		joinStrings(names, ", "),
		joinStrings(placeholders, ", "),
	)
}

func joinStrings(values []string, sep string) string {
	out := ""
	for i, value := range values {
		if i > 0 {
			out += sep
		}
		out += value
	}
	return out
}

// passwordPlain 是「待 bcrypt 哈希的明文口令」占位值。
// 静态数据规格不直接存哈希：bcrypt 每次加盐不同，哈希应在写入时生成，
// 且 dry-run 不应付出 bcrypt 计算成本。
type passwordPlain string

// insertAll 依次写入全部表。每行一次 Exec（本地 seed 数据量约 250 行，
// 逐行执行便于错误定位；Docker 内环回网络下总耗时可忽略）。
//
// 同一明文口令只算一次哈希（QA/admin/普通账号/群口令共 4 个不同明文），
// 既减少 bcrypt 计算（cost=12 时约 0.3s/次）又不影响登录（bcrypt 校验与盐无关）。
func insertAll(ctx context.Context, q persistence.Queryable, steps []tableData, bcryptCost int) error {
	hashes, err := buildPasswordHashes(steps, bcryptCost)
	if err != nil {
		return err
	}
	for _, step := range steps {
		if len(step.Rows) == 0 {
			continue
		}
		statement := step.insertSQL()
		for rowIndex, row := range step.Rows {
			if len(row) != len(step.Columns) {
				return fmt.Errorf("表 %s 第 %d 行参数个数 %d 与列数 %d 不一致", step.Table, rowIndex+1, len(row), len(step.Columns))
			}
			args := make([]any, len(row))
			for i, value := range row {
				if plain, ok := value.(passwordPlain); ok {
					hashed, ok := hashes[plain]
					if !ok {
						return fmt.Errorf("表 %s 第 %d 行的口令未预计算哈希（内部错误）", step.Table, rowIndex+1)
					}
					args[i] = hashed
					continue
				}
				args[i] = value
			}
			if err := q.Exec(ctx, statement, args...); err != nil {
				return fmt.Errorf("写入表 %s 第 %d 行失败: %w", step.Table, rowIndex+1, err)
			}
		}
	}
	return nil
}

// buildPasswordHashes 收集所有密码占位值并按明文去重计算 bcrypt 哈希。
// 哈希顺序按明文字典序固定（结果不可复现也无所谓，但便于测试断言）。
func buildPasswordHashes(steps []tableData, bcryptCost int) (map[passwordPlain]string, error) {
	uniques := make(map[passwordPlain]struct{})
	for _, step := range steps {
		for _, row := range step.Rows {
			for _, value := range row {
				if plain, ok := value.(passwordPlain); ok {
					uniques[plain] = struct{}{}
				}
			}
		}
	}
	plains := make([]string, 0, len(uniques))
	for plain := range uniques {
		plains = append(plains, string(plain))
	}
	sort.Strings(plains)
	hashes := make(map[passwordPlain]string, len(plains))
	for _, plain := range plains {
		if plain == "" {
			return nil, fmt.Errorf("口令占位值为空（内部数据规格错误）")
		}
		cost := bcryptCost
		if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
			cost = bcrypt.DefaultCost
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
		if err != nil {
			return nil, fmt.Errorf("生成口令哈希失败: %w", err)
		}
		hashes[passwordPlain(plain)] = string(hashed)
	}
	return hashes, nil
}

// countRows 统计全部表行数（dry-run 摘要）。
func countRows(steps []tableData) int {
	total := 0
	for _, step := range steps {
		total += len(step.Rows)
	}
	return total
}
