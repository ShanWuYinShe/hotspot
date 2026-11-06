package migrate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
)

// SchemaManifest 是「执行全部迁移后应得到的 schema」的最小只读快照：表、列、
// 索引名。它解决的问题是旧 runner 的启发式基线缺陷——版本表里写着「已应用」，
// 库内对象却可能不存在（历史事故：idx_moments_keyset 被基线吞掉）。
//
// 校验方向是「期望对象必须存在」（missing 致命、extra 仅告警）：允许运维手工
// 加索引/临时表，但不允许迁移声明的对象缺失。
//
// 生成：go run ./cmd/migrate -write-manifest=migrations/schema_manifest.json
// （必须在全新库上执行，保证是「迁移集合」而不是某个存量库的现状）。
// 校验：cmd/migrate -verify-only（CI/部署门禁）与每次启动 EnsureSchema 末尾。
type SchemaManifest struct {
	Tables  []string `json:"tables"`
	Columns []string `json:"columns"`
	Indexes []string `json:"indexes"`
}

const schemaManifestFileName = "schema_manifest.json"

// manifestExcludedTables 排除迁移簿记表：它们自身由 runner 维护，不参与
// 「迁移集合应产生的业务 schema」校验。
var manifestExcludedTables = map[string]struct{}{
	"schema_migrations": {},
	"data_migrations":   {},
}

// BuildSchemaManifest 从真实库读取当前 schema（排序稳定，便于 diff）。
func BuildSchemaManifest(ctx context.Context, db persistence.QueryableDB) (SchemaManifest, error) {
	manifest := SchemaManifest{Tables: []string{}, Columns: []string{}, Indexes: []string{}}

	tableRows, err := db.Query(ctx, `
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'public' AND table_type = 'BASE TABLE'`)
	if err != nil {
		return SchemaManifest{}, fmt.Errorf("list tables: %w", err)
	}
	for tableRows.Next() {
		var table string
		if err := tableRows.Scan(&table); err != nil {
			tableRows.Close()
			return SchemaManifest{}, fmt.Errorf("scan table: %w", err)
		}
		if _, excluded := manifestExcludedTables[table]; excluded {
			continue
		}
		manifest.Tables = append(manifest.Tables, table)
	}
	if err := tableRows.Err(); err != nil {
		tableRows.Close()
		return SchemaManifest{}, fmt.Errorf("list tables rows: %w", err)
	}
	tableRows.Close()

	columnRows, err := db.Query(ctx, `
SELECT table_name, column_name
FROM information_schema.columns
WHERE table_schema = 'public'`)
	if err != nil {
		return SchemaManifest{}, fmt.Errorf("list columns: %w", err)
	}
	for columnRows.Next() {
		var table, column string
		if err := columnRows.Scan(&table, &column); err != nil {
			columnRows.Close()
			return SchemaManifest{}, fmt.Errorf("scan column: %w", err)
		}
		if _, excluded := manifestExcludedTables[table]; excluded {
			continue
		}
		manifest.Columns = append(manifest.Columns, table+"."+column)
	}
	if err := columnRows.Err(); err != nil {
		columnRows.Close()
		return SchemaManifest{}, fmt.Errorf("list columns rows: %w", err)
	}
	columnRows.Close()

	indexRows, err := db.Query(ctx, `
SELECT tablename, indexname
FROM pg_indexes
WHERE schemaname = 'public'`)
	if err != nil {
		return SchemaManifest{}, fmt.Errorf("list indexes: %w", err)
	}
	for indexRows.Next() {
		var table, index string
		if err := indexRows.Scan(&table, &index); err != nil {
			indexRows.Close()
			return SchemaManifest{}, fmt.Errorf("scan index: %w", err)
		}
		if _, excluded := manifestExcludedTables[table]; excluded {
			continue
		}
		manifest.Indexes = append(manifest.Indexes, table+":"+index)
	}
	if err := indexRows.Err(); err != nil {
		indexRows.Close()
		return SchemaManifest{}, fmt.Errorf("list indexes rows: %w", err)
	}
	indexRows.Close()

	sort.Strings(manifest.Tables)
	sort.Strings(manifest.Columns)
	sort.Strings(manifest.Indexes)
	return manifest, nil
}

// Encode 序列化 manifest（缩进 JSON + 末尾换行，便于代码评审 diff）。
func (m SchemaManifest) Encode() ([]byte, error) {
	blob, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(blob, byte('\n')), nil
}

// LoadSchemaManifest 从嵌入文件系统读取 manifest。文件不存在时返回 ok=false
// （测试用的 fstest.MapFS 不携带 manifest，按「不做 manifest 校验」处理）。
func LoadSchemaManifest(fsys fs.FS) (SchemaManifest, bool, error) {
	blob, err := fs.ReadFile(fsys, schemaManifestFileName)
	if err != nil {
		if isNotExist(err) {
			return SchemaManifest{}, false, nil
		}
		return SchemaManifest{}, false, fmt.Errorf("read %s: %w", schemaManifestFileName, err)
	}
	var manifest SchemaManifest
	if err := json.Unmarshal(blob, &manifest); err != nil {
		return SchemaManifest{}, false, fmt.Errorf("parse %s: %w", schemaManifestFileName, err)
	}
	return manifest, true, nil
}

func isNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

// VerifySchemaManifest 比对期望 manifest 与库内实际 schema。返回 missing
// （期望存在但缺失的对象，致命）与 extraTables（库内多出的表，仅告警）。
func VerifySchemaManifest(ctx context.Context, db persistence.QueryableDB, want SchemaManifest) (missing []string, extraTables []string, err error) {
	got, err := BuildSchemaManifest(ctx, db)
	if err != nil {
		return nil, nil, err
	}
	missing = append(missing, diffMissing("table", want.Tables, got.Tables)...)
	missing = append(missing, diffMissing("column", want.Columns, got.Columns)...)
	missing = append(missing, diffMissing("index", want.Indexes, got.Indexes)...)
	extraTables = diffMissing("table", got.Tables, want.Tables)
	sort.Strings(missing)
	return missing, extraTables, nil
}

// diffMissing 返回 want 中不在 got 的元素（带类别前缀，便于错误信息定位）。
func diffMissing(kind string, want, got []string) []string {
	gotSet := make(map[string]struct{}, len(got))
	for _, item := range got {
		gotSet[item] = struct{}{}
	}
	missing := make([]string, 0)
	for _, item := range want {
		if _, ok := gotSet[item]; ok {
			continue
		}
		missing = append(missing, kind+":"+item)
	}
	return missing
}

// verifySchemaManifest 是 EnsureSchema/VerifySchema 的公共收尾：manifest 缺失时
// 跳过（仅测试 fstest.MapFS 路径）；缺失对象致命，多余表只告警。
func verifySchemaManifest(ctx context.Context, db persistence.QueryableDB, fsys fs.FS) error {
	manifest, ok, err := LoadSchemaManifest(fsys)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	missing, extraTables, err := VerifySchemaManifest(ctx, db, manifest)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return fmt.Errorf("schema drift: %d expected table/column/index objects are missing after applying migrations (version table says applied but schema does not match; run cmd/migrate -verify-only for details): %s",
			len(missing), strings.Join(limitStrings(missing, 25), ", "))
	}
	if len(extraTables) > 0 {
		observability.L().Warn("schema drift: tables present in database but absent from schema manifest",
			"tables", strings.Join(limitStrings(extraTables, 20), ", "))
	}
	return nil
}

// VerifyOnly 供 cmd/migrate -verify-only 使用：只读校验，不写库。
func VerifyOnly(ctx context.Context, db persistence.QueryableDB, fsys fs.FS) error {
	return VerifySchema(ctx, db, fsys)
}
