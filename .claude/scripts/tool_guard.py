#!/usr/bin/env python3
from __future__ import annotations

import json
import os
import re
import shlex
import sys
from pathlib import Path
from typing import Any

from controlled_allowlist import CONTROLLED_DEPLOY_SCRIPTS, CONTROLLED_PYTHON_SCRIPTS, INTERNAL_GUARD_SCRIPTS
from git_context import GitContextError, get_repo_context
from workflow_common import is_integration_branch

SAFE_SHELL_PATTERNS = (
    re.compile(r"^python3?\s+(?:\./)?\.claude/scripts/[a-zA-Z0-9_./-]+\.py(?:\s|$)"),
)
PY_COMPILE_PREFIX = ("python", "python3")

BLOCKED_RUNTIME_PATTERNS = (
    re.compile(r"^npm(?:\s|$)"),
    re.compile(r"^npx(?:\s|$)"),
    re.compile(r"^yarn(?:\s|$)"),
    re.compile(r"^pnpm(?:\s|$)"),
    re.compile(r"^bun(?:\s|$)"),
    re.compile(r"^node(?:\s|$)"),
    re.compile(r"^cargo(?:\s|$)"),
    re.compile(r"^rustc(?:\s|$)"),
    re.compile(r"^go(?:\s|$)"),
    re.compile(r"^python3?(?:\s|$)"),
    re.compile(r"^pip3?(?:\s|$)"),
    re.compile(r"^ohpm(?:\s|$)"),
    re.compile(r"^hvigor(?:\s|$)"),
    re.compile(r"^hvigorw(?:\s|$)"),
    re.compile(r"^docker(?:\s|$)"),
    re.compile(r"^docker-compose(?:\s|$)"),
    re.compile(r"^make(?:\s|$)"),
    re.compile(r"^wget(?:\s|$)"),
    re.compile(r"^curl(?:\s|$)"),
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
        return guard_edit_write(payload, repo)
    return 0


def deny(reason: str, blocked_input: str | None = None) -> int:
    visible_reason = reason
    if blocked_input:
        visible_reason = f"{visible_reason}\nblocked_input: {blocked_input}"
    sys.stderr.write(visible_reason + "\n")
    json.dump(
        {
            "hookSpecificOutput": {
                "hookEventName": "PreToolUse",
                "permissionDecision": "deny",
                "permissionDecisionReason": visible_reason,
            }
        },
        sys.stdout,
        ensure_ascii=False,
    )
    sys.stdout.write("\n")
    return 0


def guard_edit_write(payload: dict[str, Any], repo: dict[str, Any] | None) -> int:
    if not repo or not repo["is_main_primary"]:
        return 0

    tool_input = payload.get("tool_input", {})
    file_path = tool_input.get("file_path")
    if isinstance(file_path, str) and is_allowed_plan_file_write(file_path):
        return 0

    blocked_input = file_path if isinstance(file_path, str) else None
    return deny("主目录本地 main 禁止文件修改；请先进入 feature worktree。", blocked_input)


def is_allowed_plan_file_write(raw: str) -> bool:
    candidate = Path(os.path.expanduser(raw))
    try:
        resolved = candidate.resolve(strict=False)
    except OSError:
        return False

    plans_root = Path.home() / ".claude" / "plans"
    return is_within(resolved, plans_root)


def guard_bash(payload: dict[str, Any], repo: dict[str, Any] | None) -> int:
    command = str(payload.get("tool_input", {}).get("command", "")).strip()
    if not command:
        return 0

    if is_allowed_controlled_python(command) or is_allowed_deploy_script(command):
        return 0

    segments = [segment.strip() for segment in SPLIT_PATTERN.split(command) if segment.strip()]
    if not segments:
        return 0

    git_segments = [segment for segment in segments if is_git_segment(segment)]
    git_state_change_segments = [segment for segment in git_segments if is_git_state_change_segment(segment)]
    if len(git_segments) > 1:
        if len(git_state_change_segments) == 0:
            return deny("只读 git 查询也必须分步执行；不要使用 &&、; 或 || 串联多个命令。", command)
        return deny("git 状态变更命令必须分步执行；不要使用 &&、; 或 || 串联多个命令。", command)

    if any(is_git_segment(segment) for segment in segments):
        return guard_git_segments(segments, repo, command)

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
        if is_allowed_readonly_python_segment(argv, repo, command_cwd):
            continue
        if is_allowed_readonly_shell_segment(argv):
            continue
        if is_blocked_runtime_segment(segment):
            return deny("该命令会产生运行态副作用或依赖本地开发工具链；请优先使用 skill 指引的仓库入口，隔离运行/测试走 deploy 或受控脚本。", command)
        return deny("该命令不在允许的直接执行范围内；若需要严格流程，请优先使用 skill 暴露的仓库入口。", command)

    return 0


def guard_git_segments(segments: list[str], repo: dict[str, Any] | None, command: str) -> int:
    if repo and repo["is_main_primary"]:
        if all(is_allowed_main_git_segment(segment) for segment in segments):
            return 0
        return deny("主目录本地 main 只允许安全的 git 控制面动作；高风险历史改写、merge 兜底与破坏性清理仍然禁止。", command)

    for segment in segments:
        if not is_allowed_direct_git_segment(segment):
            return deny("当前 git 命令超出允许范围；请改用仓库 skill/脚本或先确认是否属于高风险动作。", command)
        if repo and is_integration_branch(repo.get("branch")) and is_git_merge_segment(segment):
            return deny("integration worktree 禁止使用 git merge 汇总 feature 或追平 main；请继续使用线性 history。", segment)
    return 0


def is_git_segment(segment: str) -> bool:
    return bool(re.match(r"^git(?:\s|$)", segment))


def strip_git_global_options(segment: str) -> str:
    argv = tokenize_shell_segment(segment)
    if not argv or argv[0] != "git":
        return segment
    i = 1
    while i < len(argv):
        arg = argv[i]
        if arg in {"-C", "-c", "--git-dir", "--work-tree", "--namespace", "--config-env"}:
            i += 2
            continue
        if arg in {"--no-pager", "--paginate"}:
            i += 1
            continue
        break
    rest = argv[i:]
    return "git " + " ".join(rest) if rest else "git"


def is_allowed_main_git_segment(segment: str) -> bool:
    normalized = strip_git_global_options(segment)
    return bool(
        re.match(
            r"^git\s+(?:status|diff|log|reflog|branch|rev-parse|show-ref|worktree\s+list|worktree\s+add|worktree\s+remove|rebase)(?:\s|$)",
            normalized,
        )
    ) and not is_high_risk_git_segment(normalized)


def is_allowed_direct_git_segment(segment: str) -> bool:
    normalized = strip_git_global_options(segment)
    return bool(
        re.match(
            r"^git\s+(?:status|diff|log|reflog|branch|rev-parse|show-ref|worktree\s+list|add|rm|commit|rebase|reset\s+--soft|checkout|switch|cherry-pick)(?:\s|$)",
            normalized,
        )
    ) and not is_high_risk_git_segment(normalized)


def is_high_risk_git_segment(segment: str) -> bool:
    return bool(
        re.match(
            r"^git\s+(?:reset\s+--hard|clean(?:\s|$)|merge(?:\s|$)|push\s+--force(?:-with-lease)?(?:\s|$)|branch\s+-D(?:\s|$)|checkout\s+\.|restore\s+\.|worktree\s+prune(?:\s|$))",
            segment,
        )
    )


def is_git_merge_segment(segment: str) -> bool:
    return bool(re.match(r"^git\s+merge(?:\s|$)", strip_git_global_options(segment)))


def is_git_state_change_segment(segment: str) -> bool:
    normalized = strip_git_global_options(segment)
    return bool(
        re.match(
            r"^git\s+(?:add|commit|rebase|merge|cherry-pick|worktree\s+add|worktree\s+remove|branch\s+-d|branch\s+-D|checkout|switch|reset|clean)(?:\s|$)",
            normalized,
        )
    )


def is_allowed_controlled_python(segment: str) -> bool:
    for name in (*CONTROLLED_PYTHON_SCRIPTS, *INTERNAL_GUARD_SCRIPTS):
        if re.match(rf"^python3?\s+(?:\./)?\.claude/scripts/{re.escape(name)}\.py(?:\s|$)", segment):
            return True
    return False


def is_allowed_deploy_script(segment: str) -> bool:
    for name in CONTROLLED_DEPLOY_SCRIPTS:
        if re.match(rf"^(?:\./)?deploy/scripts/{re.escape(name)}(?:\s|$)", segment):
            return True
        if re.match(rf"^bash\s+(?:\./)?deploy/scripts/{re.escape(name)}(?:\s|$)", segment):
            return True
    return False


def is_safe_shell_segment(segment: str) -> bool:
    return any(pattern.match(segment) for pattern in SAFE_SHELL_PATTERNS)


def is_allowed_readonly_shell_segment(argv: list[str]) -> bool:
    if not argv:
        return False
    cmd = argv[0]
    return cmd in {"pwd", "ls", "which", "git"}


def is_allowed_readonly_python_segment(argv: list[str], repo: dict[str, Any] | None, current_dir: Path) -> bool:
    if len(argv) < 4:
        return False
    if argv[0] not in PY_COMPILE_PREFIX:
        return False
    if argv[1:3] != ["-m", "py_compile"]:
        return False
    if not repo:
        return False
    current_root = Path(repo["top"]).resolve()
    file_args = argv[3:]
    return all_paths_within_worktree(file_args, current_root, current_dir.resolve())


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
