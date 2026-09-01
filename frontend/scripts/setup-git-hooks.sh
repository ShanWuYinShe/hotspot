#!/bin/sh
# 激活 git hooks（core.hooksPath -> 仓库根 .husky/）。
# 作为 frontend package.json 的 prepare 脚本运行：bun install 时自动调用，
# 保证 clone/install 后 pre-commit 等校验立即生效（与 CI 对齐）。
#
# 兼容性硬约束（曾致容器 crash loop）：
# - 必须 POSIX sh（#!/bin/sh），不能用 bash——frontend 运行容器无 bash
# - git 不可用/不在仓库内时静默跳过（exit 0），绝不能让 bun install 失败
# - .git 不存在（如容器内源码挂载、发布包）时跳过
#
# 不用 husky：husky 9 只探测 cwd 的 .git（不向上递归），子目录 package 的
# prepare 必然报 ".git can't be found"。这里用 git rev-parse 定位仓库根，
# 无 cwd 限制；hook 本体由 git 原生执行，不依赖任何运行时。

command -v git >/dev/null 2>&1 || exit 0
repo_root="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0
[ -f "$repo_root/.husky/pre-commit" ] || exit 0

git config core.hooksPath "$repo_root/.husky" || exit 0
echo "git hooks 已激活: core.hooksPath = $(git config core.hooksPath)"
