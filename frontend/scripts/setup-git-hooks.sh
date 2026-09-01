#!/usr/bin/env bash
# 激活 git hooks（core.hooksPath -> 仓库根 .husky/）。
# 作为 frontend package.json 的 prepare 脚本运行：bun install 时自动调用，
# 保证 clone/install 后 pre-commit 等校验立即生效（与 CI 对齐）。
#
# 不用 husky：husky 9 只探测 cwd 的 .git（不向上递归），子目录 package 的
# prepare 必然报 ".git can't be found"。这里用 git rev-parse 定位仓库根，
# 无 cwd 限制；hook 本体由 git 原生执行，不依赖任何运行时。
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
# 用绝对路径避免相对路径在不同 cwd/平台下的解析歧义（git rev-parse --git-path hooks
# 对相对 hooksPath 是相对 cwd 解析的，子目录运行 git 命令时行为易混淆）。
git config core.hooksPath "$repo_root/.husky"
echo "git hooks 已激活: core.hooksPath = $(git config core.hooksPath)"
