from __future__ import annotations

import subprocess
from pathlib import Path
from typing import Any


class GitContextError(RuntimeError):
    pass


def run_git(cwd: Path | str, *args: str) -> str:
    result = subprocess.run(
        ["git", "-C", str(cwd), *args],
        check=True,
        capture_output=True,
        text=True,
    )
    return result.stdout.strip()


def get_repo_context(cwd: Any) -> dict[str, Any]:
    if not cwd:
        raise GitContextError("缺少 cwd")

    try:
        top_level = run_git(cwd, "rev-parse", "--show-toplevel")
        common_dir = run_git(cwd, "rev-parse", "--git-common-dir")
        branch = run_git(cwd, "branch", "--show-current")
    except subprocess.CalledProcessError as exc:
        stderr = (exc.stderr or exc.stdout or "").strip()
        raise GitContextError(stderr or "无法读取 git 上下文") from exc

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
