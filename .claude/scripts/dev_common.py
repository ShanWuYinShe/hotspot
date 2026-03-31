#!/usr/bin/env python3
"""受控开发脚本的公共工具函数。

被 go_dev.py / frontend_dev.py / harmony_worktree.py 共同引用。
"""
from __future__ import annotations

import json
import subprocess
import sys
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Sequence

from git_context import GitContextError, get_repo_context


@dataclass
class CommandResult:
    argv: list[str]
    returncode: int
    stdout: str
    stderr: str


def now_stamp() -> str:
    return datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")


def run(argv: Sequence[str], cwd: Path) -> CommandResult:
    completed = subprocess.run(list(argv), cwd=str(cwd), capture_output=True, text=True)
    return CommandResult(list(argv), completed.returncode, completed.stdout, completed.stderr)


def write_command_log(path: Path, result: CommandResult) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    body = [
        "$ " + " ".join(result.argv),
        "",
        "[stdout]",
        result.stdout.rstrip(),
        "",
        "[stderr]",
        result.stderr.rstrip(),
        "",
        f"[exit] {result.returncode}",
    ]
    path.write_text("\n".join(body).rstrip() + "\n", encoding="utf-8")


def summarize_failure(result: CommandResult, keywords: tuple[str, ...] = ("error", "failed", "exception")) -> list[str]:
    lines: list[str] = []
    for source in (result.stderr, result.stdout):
        for line in source.splitlines():
            stripped = line.strip()
            if not stripped:
                continue
            lowered = stripped.lower()
            if any(token in lowered for token in keywords):
                lines.append(stripped)
    deduped: list[str] = []
    seen: set[str] = set()
    for line in lines:
        if line in seen:
            continue
        seen.add(line)
        deduped.append(line)
    return deduped[:20]


def ensure_allowed_worktree(cwd: Path) -> dict:
    try:
        repo = get_repo_context(cwd)
    except GitContextError as exc:
        raise RuntimeError(f"无法读取 git 上下文: {exc}") from exc
    if repo["is_main_primary"]:
        raise RuntimeError("受控脚本只能在非主目录 worktree 中运行，不能在主目录本地 main 上运行。")
    return repo


def ensure_project_dir(repo_top: Path, subdir: str, marker: str, explicit: str | None = None) -> Path:
    target = Path(explicit).expanduser().resolve() if explicit else (repo_top / subdir).resolve()
    if not target.is_dir():
        raise RuntimeError(f"工程目录不存在: {target}")
    marker_file = target / marker
    if not marker_file.is_file():
        raise RuntimeError(f"未找到 {marker}: {marker_file}")
    return target


def finish_report(report: dict, logs_dir: Path) -> int:
    json.dump(report, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0 if report.get("status") == "passed" else 2


def finish_blocked(task: str, reason: str, logs_dir: Path) -> int:
    json.dump({"status": "blocked", "task": task, "reason": reason, "logs_dir": str(logs_dir)}, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 1
