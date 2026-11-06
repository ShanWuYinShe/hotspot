package migrations

import "embed"

// Files 嵌入全部迁移 SQL 与 schema manifest（cmd/migrate -write-manifest 生成，
// 随迁移集合一起评审：manifest 变更 = schema 变更）。
//
//go:embed *.sql schema_manifest.json
var Files embed.FS
