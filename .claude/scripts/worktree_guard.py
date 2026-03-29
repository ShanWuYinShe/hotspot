#!/usr/bin/env python3
import json
import re
import subprocess
import sys
from pathlib import Path

from git_context import GitContextError, get_repo_context, run_git

_VALID_SUBPROJECTS = frozenset({
    "backend",
    "backend-go",
    "frontend-svelte",
    "harmonyapp",
    "spec",
    "deploy",
    "claude-config",
})

_WORKTREE_NAME_RE = re.compile(
    r"^(?P<subproject>[a-z][a-z0-9-]+)\.(?P<module>[a-z][a-z0-9-]+)\.(?P<feature>[a-z][a-z0-9-]+)$"
)


def validate_worktree_name(name: str) -> str | None:
    """Return an error message if the name is invalid, or None if valid."""
    m = _WORKTREE_NAME_RE.match(name)
    if not m:
        return (
            f"worktree 名称 '{name}' 不符合命名规则。"
            f"必须使用 '{{子项目名}}.{{模块名}}.{{功能描述}}' 格式，"
            f"各段仅允许小写字母、数字和连字符，以字母开头。"
            f"\n示例: backend-go.auth.add-oauth, harmonyapp.profile.fix-padding"
        )
    subproject = m.group("subproject")
    if subproject not in _VALID_SUBPROJECTS:
        valid = ", ".join(sorted(_VALID_SUBPROJECTS))
        return f"未知子项目 '{subproject}'，当前允许: {valid}"
    return None


def fail(message: str) -> int:
    sys.stderr.write(message + "\n")
    return 1


def main() -> int:
    try:
        payload = json.load(sys.stdin)
    except Exception as exc:
        return fail(f"WorktreeCreate hook 无法解析输入: {exc}")

    cwd_raw = payload.get("cwd")
    if not cwd_raw:
        return fail("WorktreeCreate hook 缺少 cwd，无法确认主目录。")

    cwd = Path(str(cwd_raw)).resolve()

    try:
        repo = get_repo_context(cwd)
        status = run_git(cwd, "status", "--porcelain")
        run_git(cwd, "show-ref", "--verify", "refs/heads/main")
    except GitContextError as exc:
        return fail(f"WorktreeCreate hook 无法读取 git 上下文: {exc}")
    except subprocess.CalledProcessError as exc:
        stderr = (exc.stderr or exc.stdout or "").strip()
        return fail(f"WorktreeCreate hook 无法读取 git 状态: {stderr}")

    primary_root = Path(repo["primary_root"])

    if not repo["is_primary_checkout"]:
        return fail("EnterWorktree 只能从主目录执行，当前不在主目录 checkout。")

    if repo["branch"] != "main":
        return fail("EnterWorktree 只能从主目录本地 main 分支执行。")

    if status:
        return fail("EnterWorktree 前主目录必须保持本地 main 且无未提交改动。")

    name = str(payload.get("name") or "").strip()
    if not name:
        return fail(
            "必须为 worktree 指定名称，格式为 '{子项目名}.{模块名}.{功能描述}'。"
            "\n示例: backend-go.auth.add-oauth, harmonyapp.profile.fix-padding"
        )

    name_error = validate_worktree_name(name)
    if name_error:
        return fail(name_error)

    worktree_dir = primary_root / ".claude" / "worktrees" / name
    branch_name = f"worktree-{name}"

    try:
        subprocess.run(
            ["git", "-C", str(primary_root), "worktree", "add", "-b", branch_name, str(worktree_dir), "main"],
            check=True,
            capture_output=True,
            text=True,
        )
    except subprocess.CalledProcessError as exc:
        stderr = (exc.stderr or exc.stdout or "").strip()
        return fail(f"基于本地 main 创建 worktree 失败: {stderr}")

    sys.stdout.write(str(worktree_dir.resolve()) + "\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
