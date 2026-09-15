// Package buildinfo 承载构建期注入的版本信息（-ldflags -X 注入，见
// backend-go/Dockerfile 与根 scripts/deploy-production.sh）。零值兜底为
// "dev"，本地 go build / go run 不注入也能明确区分于线上产物——线上排障时
// health 接口与启动日志里的 version/commit 可精确回溯到构建源码。
package buildinfo

import "time"

// 注入点（-ldflags "-X hotspot/backend-go/internal/platform/buildinfo.<var>=..."）。
var (
	// Version 人类可读版本号：git describe --tags --always --dirty 的产物，
	// 如 v1.0.0 / v1.0.0-2-g1a2b3c4；无 tag 时为裸短 sha。
	Version = "dev"
	// Commit 构建对应源码的完整 git commit sha，版本争议时的最终裁决依据。
	Commit = ""
	// BuildTime 镜像构建时刻（UTC RFC3339），区分同一 commit 的多次构建。
	BuildTime = ""
)

// Committed 判定当前产物是否经过正式构建注入（非本地 dev 裸构建）。
func Committed() bool { return Version != "dev" || Commit != "" }

// Summary 返回单行版本摘要（启动日志用），dev 裸构建只返回 dev。
func Summary() string {
	if !Committed() {
		return "dev"
	}
	s := Version
	if Commit != "" {
		s += " (commit " + Commit + ")"
	}
	if BuildTime != "" {
		if _, err := time.Parse(time.RFC3339, BuildTime); err == nil {
			s += " built " + BuildTime
		}
	}
	return s
}
