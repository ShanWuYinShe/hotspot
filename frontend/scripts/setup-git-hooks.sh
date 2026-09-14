#!/bin/sh
# 激活 git hooks（core.hooksPath -> 主工作树的 .husky/）。
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
#
# 为什么用 --git-common-dir 而不是 --show-toplevel（曾致 hooks 静默失效）：
# core.hooksPath 存在**各 worktree 共享**的 .git/config 里，而
# `git rev-parse --show-toplevel` 在 worktree 内返回该 worktree 自身的路径。
# 于是任何 worktree 里跑一次 bun install，都会把全仓的 core.hooksPath 改写成
# 该 worktree 的 .husky；worktree 一旦删除，路径失效，所有提交的 pre-commit
# 校验静默消失（git 对不存在的 hooksPath 不报错）。改用 --git-common-dir 得到
# 共享 git 目录（worktree 内恒为主仓库的 .git），其父目录即主工作树根，
# 所有 worktree 写入同一稳定路径。hook 执行时 cwd 为提交所在工作树根，
# 因此 `bun run --cwd frontend check` 仍校验提交所在的那份代码。

command -v git >/dev/null 2>&1 || exit 0

common_dir="$(git rev-parse --git-common-dir 2>/dev/null)" || exit 0
[ -n "$common_dir" ] || exit 0
# --git-common-dir 在主工作树返回相对路径（.git），在 worktree 返回绝对路径；
# 统一解析为绝对路径后再取父目录。
case "$common_dir" in
  /*) ;;
  *) common_dir="$(cd "$common_dir" 2>/dev/null && pwd)" || exit 0 ;;
esac
[ -n "$common_dir" ] || exit 0
repo_root="$(dirname "$common_dir")"

[ -f "$repo_root/.husky/pre-commit" ] || exit 0

git config core.hooksPath "$repo_root/.husky" || exit 0
echo "git hooks 已激活: core.hooksPath = $(git config core.hooksPath)"
