#!/usr/bin/env python3
"""在当前 worktree 中受控运行 Go 格式化 / 语法检查 / 编译。

只放行不影响宿主环境的静态操作：fmt、vet、build。
测试（go test）不在本脚本范围内，必须通过 deploy/ 隔离执行。

用法：
    python3 .claude/scripts/go_dev.py --task fmt
    python3 .claude/scripts/go_dev.py --task vet
    python3 .claude/scripts/go_dev.py --task build
    python3 .claude/scripts/go_dev.py --task all        # fmt + vet + build
    python3 .claude/scripts/go_dev.py --task check       # 同 all
"""
from __future__ import annotations

import argparse
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


def summarize_failure(result: CommandResult) -> list[str]:
    lines: list[str] = []
    for source in (result.stderr, result.stdout):
        for line in source.splitlines():
            stripped = line.strip()
            if not stripped:
                continue
            lowered = stripped.lower()
            if any(token in lowered for token in ("error", "failed", "exception")):
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
        raise RuntimeError("go_dev.py 只能在非主目录 worktree 中运行，不能在主目录本地 main 上运行。")
    return repo


def detect_go_root(repo_top: Path, explicit: str | None) -> Path:
    target = Path(explicit).expanduser().resolve() if explicit else (repo_top / "backend-go").resolve()
    if not target.is_dir():
        raise RuntimeError(f"Go 工程目录不存在: {target}")
    go_mod = target / "go.mod"
    if not go_mod.is_file():
        raise RuntimeError(f"未找到 go.mod: {go_mod}")
    return target


ALLOWED_TASKS = {"fmt", "vet", "build", "check", "all"}


def run_stage(go_root: Path, logs_dir: Path, stage: str) -> tuple[int, dict]:
    if stage == "fmt":
        argv = ["go", "fmt", "./..."]
    elif stage == "vet":
        argv = ["go", "vet", "./..."]
    elif stage == "build":
        argv = ["go", "build", "./..."]
    else:
        raise RuntimeError(f"未知阶段: {stage}")

    result = run(argv, go_root)
    log_path = logs_dir / f"{stage}.log"
    write_command_log(log_path, result)

    if result.returncode == 0:
        return 0, {
            "status": "passed",
            "stage": stage,
            "go_root": str(go_root),
            "command": argv,
            "log": str(log_path),
            "summary": [f"Go {stage} 通过。"],
        }
    return 2, {
        "status": "failed",
        "stage": stage,
        "go_root": str(go_root),
        "command": argv,
        "log": str(log_path),
        "summary": summarize_failure(result),
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="在当前 worktree 中受控运行 Go fmt/vet/build")
    parser.add_argument("--go-root", help="Go 工程目录，默认 <repo>/backend-go")
    parser.add_argument("--task", choices=sorted(ALLOWED_TASKS), default="check")
    args = parser.parse_args()

    cwd = Path.cwd().resolve()
    logs_dir = cwd / "tmp" / "go-dev" / now_stamp()
    logs_dir.mkdir(parents=True, exist_ok=True)

    try:
        repo = ensure_allowed_worktree(cwd)
        repo_top = Path(repo["top"]).resolve()
        go_root = detect_go_root(repo_top, args.go_root)

        stages: list[str]
        if args.task in {"check", "all"}:
            stages = ["fmt", "vet", "build"]
        else:
            stages = [args.task]

        report: dict[str, object] = {
            "status": "passed",
            "task": args.task,
            "worktree_root": str(repo_top),
            "go_root": str(go_root),
            "logs_dir": str(logs_dir),
            "steps": [],
        }

        for stage in stages:
            code, payload = run_stage(go_root, logs_dir, stage)
            report["steps"].append(payload)
            if payload["status"] == "failed":
                report["status"] = "failed"
                report["summary"] = payload["summary"]
                json.dump(report, sys.stdout, ensure_ascii=False, indent=2)
                sys.stdout.write("\n")
                return code

        report["summary"] = [f"Go {args.task} 全部通过。"]
        json.dump(report, sys.stdout, ensure_ascii=False, indent=2)
        sys.stdout.write("\n")
        return 0

    except RuntimeError as exc:
        payload = {
            "status": "blocked",
            "task": args.task,
            "reason": str(exc),
            "logs_dir": str(logs_dir),
        }
        json.dump(payload, sys.stdout, ensure_ascii=False, indent=2)
        sys.stdout.write("\n")
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
