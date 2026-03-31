#!/usr/bin/env python3
"""在当前 worktree 中受控运行前端格式化 / 语法检查 / 编译。

只放行不影响宿主环境的静态操作：install、check、lint、svelte-check、build。
测试（pnpm test）不在本脚本范围内，必须通过 deploy/ 隔离执行。

用法：
    python3 .claude/scripts/frontend_dev.py --task install
    python3 .claude/scripts/frontend_dev.py --task check
    python3 .claude/scripts/frontend_dev.py --task lint
    python3 .claude/scripts/frontend_dev.py --task build
    python3 .claude/scripts/frontend_dev.py --task all        # install + check + lint + build
"""
from __future__ import annotations

import argparse
import shutil
from pathlib import Path

from dev_common import (
    ensure_allowed_worktree,
    ensure_project_dir,
    finish_blocked,
    finish_report,
    now_stamp,
    run,
    summarize_failure,
    write_command_log,
)

_FAILURE_KEYWORDS = ("error", "failed", "exception", "warning")


def detect_pnpm_command() -> str:
    found = shutil.which("pnpm")
    if found:
        return found
    raise RuntimeError("未找到 pnpm。请先安装 pnpm。")


STAGE_COMMANDS = {
    "install": lambda pnpm: [pnpm, "install"],
    "check": lambda pnpm: [pnpm, "check"],
    "lint": lambda pnpm: [pnpm, "lint"],
    "svelte-check": lambda pnpm: [pnpm, "exec", "svelte-check"],
    "build": lambda pnpm: [pnpm, "build"],
}

ALLOWED_TASKS = {"install", "check", "lint", "svelte-check", "build", "all"}


def run_stage(frontend_root: Path, logs_dir: Path, stage: str, pnpm: str) -> tuple[int, dict]:
    command_fn = STAGE_COMMANDS.get(stage)
    if command_fn is None:
        raise RuntimeError(f"未知阶段: {stage}")

    argv = command_fn(pnpm)
    result = run(argv, frontend_root)
    log_path = logs_dir / f"{stage}.log"
    write_command_log(log_path, result)

    if result.returncode == 0:
        return 0, {
            "status": "passed",
            "stage": stage,
            "frontend_root": str(frontend_root),
            "command": argv,
            "log": str(log_path),
            "summary": [f"前端 {stage} 通过。"],
        }
    return 2, {
        "status": "failed",
        "stage": stage,
        "frontend_root": str(frontend_root),
        "command": argv,
        "log": str(log_path),
        "summary": summarize_failure(result, keywords=_FAILURE_KEYWORDS),
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="在当前 worktree 中受控运行前端 install/check/lint/build")
    parser.add_argument("--frontend-root", help="前端工程目录，默认 <repo>/frontend_svelte")
    parser.add_argument("--task", choices=sorted(ALLOWED_TASKS), default="check")
    args = parser.parse_args()

    cwd = Path.cwd().resolve()
    logs_dir = cwd / "tmp" / "frontend-dev" / now_stamp()
    logs_dir.mkdir(parents=True, exist_ok=True)

    try:
        repo = ensure_allowed_worktree(cwd)
        repo_top = Path(repo["top"]).resolve()
        frontend_root = ensure_project_dir(repo_top, "frontend_svelte", "package.json", args.frontend_root)
        pnpm = detect_pnpm_command()

        stages: list[str]
        if args.task == "all":
            stages = ["install", "check", "lint", "build"]
        else:
            stages = [args.task]

        report: dict[str, object] = {
            "status": "passed",
            "task": args.task,
            "worktree_root": str(repo_top),
            "frontend_root": str(frontend_root),
            "logs_dir": str(logs_dir),
            "steps": [],
        }

        for stage in stages:
            code, payload = run_stage(frontend_root, logs_dir, stage, pnpm)
            report["steps"].append(payload)
            if payload["status"] == "failed":
                report["status"] = "failed"
                report["summary"] = payload["summary"]
                return finish_report(report, logs_dir)

        report["summary"] = [f"前端 {args.task} 全部通过。"]
        return finish_report(report, logs_dir)

    except RuntimeError as exc:
        return finish_blocked(args.task, str(exc), logs_dir)


if __name__ == "__main__":
    raise SystemExit(main())
