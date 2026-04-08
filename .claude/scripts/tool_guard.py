#!/usr/bin/env python3
import json
import os
import re
import shlex
import sys
from pathlib import Path
from typing import Any

from controlled_allowlist import CONTROLLED_DEPLOY_SCRIPTS, CONTROLLED_PYTHON_SCRIPTS
from git_context import GitContextError, get_repo_context


SAFE_SHELL_PATTERNS = (
    re.compile(r"^python3?\s+(?:\./)?\.claude/scripts/[a-zA-Z0-9_./-]+\.py(?:\s|$)"),
)

BLOCKED_RUNTIME_PATTERNS = (
    re.compile(r"^git(?:\s|$)"),
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
        return guard_edit_write(payload, repo)
    return 0


def deny(reason: str, blocked_input: str | None = None) -> int:
    if blocked_input:
        reason = f"{reason}\nblocked_input: {blocked_input}"
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

    if repo and repo["is_main_primary"]:
        if all(is_allowed_direct_git_read_segment(segment) for segment in segments):
            return 0
        if all(is_allowed_controlled_python(segment) or is_allowed_deploy_script(segment) for segment in segments):
            return 0
        return deny("主目录本地 main 只允许只读 git 查询或受控脚本入口。", command)

    if any(is_git_segment(segment) for segment in segments):
        if all(is_allowed_direct_git_segment(segment) for segment in segments):
            return 0
        return deny("当前 git 命令不在允许范围内。若该步骤涉及复杂历史整理、冲突处理或删除，请切到 Bash 模式手动分步执行。", command)

    if repo and is_integration_branch(repo.get("branch")):
        for segment in segments:
            if is_git_merge_segment(segment):
                return deny("integration worktree 禁止使用 git merge 汇总 feature 或追平 main；请改用受控脚本中的 cherry-pick / replay / rebase / squash / refresh-rebuild 入口。", segment)

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
            return deny("非 git 控制面命令请通过 deploy/ 入口或 .claude/scripts/ 受控脚本执行。", command)
        return deny("非 git 控制面命令请通过 deploy/ 入口或 .claude/scripts/ 受控脚本执行。", command)

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


def is_allowed_direct_git_read_segment(segment: str) -> bool:
    normalized = strip_git_global_options(segment)
    return bool(
        re.match(
            r"^git\s+(?:status|diff|log|branch|rev-parse|show-ref|worktree\s+list)(?:\s|$)",
            normalized,
        )
    )


def is_allowed_direct_git_segment(segment: str) -> bool:
    normalized = strip_git_global_options(segment)
    return bool(
        re.match(
            r"^git\s+(?:status|diff|log|branch|rev-parse|show-ref|worktree\s+list|add|rm|commit|rebase)(?:\s|$)",
            normalized,
        )
    )


def is_git_merge_segment(segment: str) -> bool:
    return bool(re.match(r"^git\s+merge(?:\s|$)", segment))


def is_git_state_change_segment(segment: str) -> bool:
    return bool(
        re.match(
            r"^git\s+(?:add|commit|rebase|merge|cherry-pick|worktree\s+add|worktree\s+remove|branch\s+-d|branch\s+-D|checkout|switch|reset|clean)(?:\s|$)",
            segment,
        )
    )


def is_integration_branch(branch: Any) -> bool:
    return isinstance(branch, str) and branch.startswith("integration-")


def is_allowed_controlled_python(segment: str) -> bool:
    for name in CONTROLLED_PYTHON_SCRIPTS:
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
