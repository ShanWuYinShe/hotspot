#!/usr/bin/env python3
import json
import os
import re
import shlex
import sys
from pathlib import Path
from typing import Any

from git_context import GitContextError, get_repo_context


SAFE_SHELL_PATTERNS = (
    re.compile(r"^git(?:\s|$)"),
    re.compile(r"^pwd(?:\s|$)"),
    re.compile(r"^ls(?:\s|$)"),
    re.compile(r"^chmod(?:\s|$)"),
    re.compile(r"^sleep(?:\s|$)"),
    re.compile(r"^curl(?:\s|$)"),
    re.compile(r"^(?:\./)?deploy/scripts/[a-z0-9-]+\.sh(?:\s|$)"),
    re.compile(r"^bash\s+(?:\./)?deploy/scripts/[a-z0-9-]+\.sh(?:\s|$)"),
    re.compile(r"^python3?\s+(?:\./)?\.claude/scripts/[a-zA-Z0-9_./-]+(?:\s|$)"),
)

BLOCKED_RUNTIME_PATTERNS = (
    re.compile(r"^npm(?:\s|$)"),
    re.compile(r"^npx(?:\s|$)"),
    re.compile(r"^node(?:\s|$)"),
    re.compile(r"^bun(?:\s|$)"),
    re.compile(r"^cargo(?:\s|$)"),
    re.compile(r"^rustc(?:\s|$)"),
    re.compile(r"^docker(?:\s|$)"),
    re.compile(r"^docker-compose(?:\s|$)"),
    re.compile(r"^go(?:\s|$)"),
    re.compile(r"^pnpm(?:\s|$)"),
    re.compile(r"^python3?(?:\s|$)"),
    re.compile(r"^ohpm(?:\s|$)"),
    re.compile(r"^hvigor(?:\s|$)"),
    re.compile(r"^hvigorw(?:\s|$)"),
    re.compile(r"^\./(?!deploy/scripts/|\.claude/scripts/)[^\s]+"),
)

SHELL_EDIT_COMMANDS = {"sed"}
SED_FILE_OPTIONS = {"-f", "--file"}
SPLIT_PATTERN = re.compile(r"\s*(?:&&|\|\||;)\s*")


def main() -> int:
    try:
        payload = json.load(sys.stdin)
    except Exception:
        return 0

    tool_name = str(payload.get("tool_name", ""))
    cwd = payload.get("cwd")
    try:
        repo = get_repo_context(cwd)
    except GitContextError:
        repo = None

    if tool_name == "Bash":
        return guard_bash(payload, repo)
    if tool_name in {"Edit", "Write"}:
        return guard_write_like(payload, repo)
    return 0


def deny(reason: str) -> int:
    json.dump(
        {
            "hookSpecificOutput": {
                "hookEventName": "PreToolUse",
                "permissionDecision": "deny",
                "permissionDecisionReason": reason,
            }
        },
        sys.stdout,
        ensure_ascii=False,
    )
    sys.stdout.write("\n")
    return 0


def guard_bash(payload: dict[str, Any], repo: dict[str, Any] | None) -> int:
    command = str(payload.get("tool_input", {}).get("command", "")).strip()
    if not command:
        return 0

    segments = [segment.strip() for segment in SPLIT_PATTERN.split(command) if segment.strip()]
    if not segments:
        return 0

    for segment in segments:
        if re.match(r"^git\s+worktree\s+add", segment) and re.search(r"\borigin/", segment):
            return deny("git worktree add 不允许使用远程引用（如 origin/main）作为基准点；必须使用本地分支引用（如 main），确保基于主目录本地 main 当前 HEAD 创建。")

    if repo and repo["is_main_primary"]:
        if all(is_git_segment(segment) for segment in segments):
            return 0
        return deny("主目录本地 main 只允许 git 控制面命令；请先进入 feature worktree 再实施、验证或运行脚本。")

    raw_cwd = payload.get("cwd")
    if raw_cwd:
        command_cwd = Path(str(raw_cwd)).resolve()
    elif repo:
        command_cwd = Path(repo["top"]).resolve()
    else:
        command_cwd = Path.cwd().resolve()

    for segment in segments:
        argv = tokenize_shell_segment(segment)
        if is_allowed_worktree_segment(argv, repo, command_cwd):
            continue
        if is_safe_shell_segment(segment):
            continue
        if is_blocked_runtime_segment(segment):
            return deny("该命令会影响运行环境；请改用 deploy/ 标准入口或 .claude/scripts/ 受控脚本。")
        return deny("当前仓库 Bash 默认只允许 git 控制面、deploy 入口、.claude/scripts/ 受控脚本，以及 worktree 内的 sed 就地编辑。开发命令（go/pnpm/ohpm/hvigor）请使用 .claude/scripts/ 下的受控脚本。")

    return 0


def guard_write_like(payload: dict[str, Any], repo: dict[str, Any] | None) -> int:
    if not repo:
        return 0

    raw_path = payload.get("tool_input", {}).get("file_path")
    if not raw_path:
        return 0

    target = Path(str(raw_path)).resolve()
    current_root = Path(repo["top"]).resolve()
    primary_root = Path(repo["primary_root"]).resolve()

    if not is_within(target, primary_root):
        return 0

    if repo["is_main_primary"]:
        return deny("主目录本地 main 禁止承载实现性文件修改；请先进入 feature worktree。")

    if not is_within(target, current_root):
        return deny("当前会话只允许修改当前 checkout / worktree 内的文件；不要跨 worktree 或回写主目录文件。")

    return 0


def is_git_segment(segment: str) -> bool:
    return bool(re.match(r"^git(?:\s|$)", segment))


def is_safe_shell_segment(segment: str) -> bool:
    return any(pattern.match(segment) for pattern in SAFE_SHELL_PATTERNS)


def is_blocked_runtime_segment(segment: str) -> bool:
    allowed_python = re.match(r"^python3?\s+(?:\./)?\.claude/scripts/[a-zA-Z0-9_./-]+(?:\s|$)", segment)
    if allowed_python:
        return False
    return any(pattern.match(segment) for pattern in BLOCKED_RUNTIME_PATTERNS)


def is_allowed_worktree_segment(argv: list[str], repo: dict[str, Any] | None, current_dir: Path) -> bool:
    if not argv or not repo or repo["is_main_primary"]:
        return False

    current_root = Path(repo["top"]).resolve()
    current_dir = current_dir.resolve()
    if not is_within(current_dir, current_root):
        return False

    if is_allowed_shell_edit(argv, current_root, current_dir):
        return True
    return False


def is_allowed_shell_edit(argv: list[str], current_root: Path, current_dir: Path) -> bool:
    cmd = argv[0]
    if cmd not in SHELL_EDIT_COMMANDS:
        return False

    file_args = extract_shell_edit_file_args(argv)
    if not file_args:
        return False
    return all_paths_within_worktree(file_args, current_root, current_dir)


def tokenize_shell_segment(segment: str) -> list[str]:
    try:
        return shlex.split(segment)
    except ValueError:
        return segment.split()


def extract_shell_edit_file_args(argv: list[str]) -> list[str]:
    option_with_value = SED_FILE_OPTIONS
    files: list[str] = []
    skip_next = False
    consumed_program = False
    for index, arg in enumerate(argv[1:], start=1):
        if skip_next:
            skip_next = False
            continue
        if arg in option_with_value:
            if index + 1 >= len(argv):
                return []
            files.append(argv[index + 1])
            skip_next = True
            continue
        if arg.startswith("-"):
            continue
        if not consumed_program:
            consumed_program = True
            continue
        files.append(arg)
    return files


def all_paths_within_worktree(args: list[str], current_root: Path, current_dir: Path) -> bool:
    for arg in args:
        resolved = resolve_path_in_worktree(arg, current_root, current_dir)
        if resolved is None:
            return False
    return True


def resolve_path_in_worktree(raw: str, current_root: Path, current_dir: Path) -> Path | None:
    expanded = Path(os.path.expanduser(raw))
    candidate = expanded if expanded.is_absolute() else (current_dir / expanded)
    try:
        resolved = candidate.resolve(strict=False)
    except OSError:
        return None
    if not is_within(resolved, current_root):
        return None
    return resolved


def is_within(path: Path, base: Path) -> bool:
    try:
        path.relative_to(base)
        return True
    except ValueError:
        return False


if __name__ == "__main__":
    raise SystemExit(main())
