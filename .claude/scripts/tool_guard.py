#!/usr/bin/env python3
import json
import re
import shlex
import subprocess
import sys
from pathlib import Path
from typing import Any


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
    re.compile(r"^python3?(?:\s|$)"),
    re.compile(r"^\./(?!deploy/scripts/|\.claude/scripts/)[^\s]+"),
)

SPLIT_PATTERN = re.compile(r"\s*(?:&&|\|\||;)\s*")


def main() -> int:
    try:
        payload = json.load(sys.stdin)
    except Exception:
        return 0

    tool_name = str(payload.get("tool_name", ""))
    cwd = payload.get("cwd")
    repo = get_repo_context(cwd)

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


def get_repo_context(cwd: Any) -> dict[str, Any] | None:
    if not cwd:
        return None
    try:
        top_level = run_git(cwd, "rev-parse", "--show-toplevel")
        common_dir = run_git(cwd, "rev-parse", "--git-common-dir")
        branch = run_git(cwd, "branch", "--show-current")
    except Exception:
        return None

    top_path = Path(top_level).resolve()
    common_path = Path(common_dir)
    if not common_path.is_absolute():
        common_path = (Path(cwd) / common_path).resolve()
    else:
        common_path = common_path.resolve()

    primary_root = common_path.parent if common_path.name == ".git" else top_path
    primary_root = primary_root.resolve()

    return {
        "cwd": Path(cwd).resolve(),
        "top": top_path,
        "branch": branch,
        "primary_root": primary_root,
        "is_primary_checkout": top_path == primary_root,
        "is_main_primary": top_path == primary_root and branch == "main",
    }


def run_git(cwd: Any, *args: str) -> str:
    result = subprocess.run(
        ["git", "-C", str(cwd), *args],
        check=True,
        capture_output=True,
        text=True,
    )
    return result.stdout.strip()


def guard_bash(payload: dict[str, Any], repo: dict[str, Any] | None) -> int:
    command = str(payload.get("tool_input", {}).get("command", "")).strip()
    if not command:
        return 0

    segments = [segment.strip() for segment in SPLIT_PATTERN.split(command) if segment.strip()]
    if not segments:
        return 0

    # 拦截使用远程引用作为基准点的 git worktree add
    for segment in segments:
        if re.match(r"^git\s+worktree\s+add", segment) and re.search(r"\borigin/", segment):
            return deny("git worktree add 不允许使用远程引用（如 origin/main）作为基准点；必须使用本地分支引用（如 main），确保基于主目录本地 main 当前 HEAD 创建。")

    if repo and repo["is_main_primary"]:
        if all(is_git_segment(segment) for segment in segments):
            return 0
        return deny("主目录本地 main 只允许 git 控制面命令；请先进入 feature worktree 再实施、验证或运行脚本。")

    for segment in segments:
        if is_blocked_runtime_segment(segment):
            return deny("该命令会影响运行环境、产生较重副作用，或可能在多 agent 下冲突；请改用 deploy/ 标准入口或仓库控制脚本。纯静态检查可直接宿主执行。")
        if not is_safe_shell_segment(segment):
            return deny("当前仓库的 Bash 默认只允许 git 控制面、deploy 入口、.claude 控制脚本，以及按副作用判定为安全的静态检查；其他命令请改用专用工具或受控脚本。")

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
    if any(pattern.match(segment) for pattern in SAFE_SHELL_PATTERNS):
        return True
    argv = tokenize_shell_segment(segment)
    if not argv:
        return False
    return is_safe_static_check(argv)


def is_blocked_runtime_segment(segment: str) -> bool:
    allowed_python = re.match(r"^python3?\s+(?:\./)?\.claude/scripts/[a-zA-Z0-9_./-]+(?:\s|$)", segment)
    if allowed_python:
        return False

    argv = tokenize_shell_segment(segment)
    if not argv:
        return False

    if is_safe_static_check(argv):
        return False
    if is_environment_affecting_command(argv):
        return True

    return any(pattern.match(segment) for pattern in BLOCKED_RUNTIME_PATTERNS)


def tokenize_shell_segment(segment: str) -> list[str]:
    try:
        return shlex.split(segment)
    except ValueError:
        return segment.split()


def is_safe_static_check(argv: list[str]) -> bool:
    if not argv:
        return False

    cmd = argv[0]
    meaningful = filtered_command_args(cmd, argv[1:])

    if cmd == "pnpm":
        if meaningful and meaningful[-1] in {"check", "lint"}:
            return True
        if len(meaningful) >= 2 and meaningful[-2:] == ["exec", "svelte-check"]:
            return True

    if cmd == "go":
        if meaningful and meaningful[0] in {"vet", "fmt"}:
            return True

    return False


def is_environment_affecting_command(argv: list[str]) -> bool:
    if not argv:
        return False

    cmd = argv[0]
    meaningful = filtered_command_args(cmd, argv[1:])

    if cmd == "pnpm" and meaningful:
        return meaningful[-1] in {"dev", "build", "test", "preview", "install", "add", "remove", "up", "dlx"}
    if cmd == "go" and meaningful:
        return meaningful[0] in {"test", "run", "build", "install"}

    return False


def filtered_command_args(cmd: str, args: list[str]) -> list[str]:
    option_with_value = {
        "pnpm": {"--dir", "-C", "--filter", "-F", "--config"},
        "go": {"-C", "-modfile", "-overlay", "-toolexec"},
    }.get(cmd, set())

    meaningful: list[str] = []
    skip_next = False
    for arg in args:
        if skip_next:
            skip_next = False
            continue
        if arg in option_with_value:
            skip_next = True
            continue
        if arg.startswith("-"):
            continue
        meaningful.append(arg)
    return meaningful


def is_within(path: Path, base: Path) -> bool:
    try:
        path.relative_to(base)
        return True
    except ValueError:
        return False


if __name__ == "__main__":
    raise SystemExit(main())
