#!/usr/bin/env python3
"""在当前 worktree 中受控运行 Rust 格式化 / 语法检查 / lint / 编译。

只放行不影响宿主环境的静态操作：fmt、check、clippy、build。
测试（cargo test）不在本脚本范围内，必须通过 deploy/ 隔离执行。

用法：
    python3 .claude/scripts/rust_dev.py --task fmt
    python3 .claude/scripts/rust_dev.py --task check
    python3 .claude/scripts/rust_dev.py --task clippy
    python3 .claude/scripts/rust_dev.py --task build
    python3 .claude/scripts/rust_dev.py --task all        # fmt + check + clippy + build
"""
from __future__ import annotations

import argparse
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


ALLOWED_TASKS = {"fmt", "check", "clippy", "build", "all"}

STAGE_ARGV = {
    "fmt": ["cargo", "fmt", "--all", "--", "--check"],
    "check": ["cargo", "check", "--all"],
    "clippy": ["cargo", "clippy", "--all", "--", "-D", "warnings"],
    "build": ["cargo", "build", "--all"],
}


def run_stage(rust_root: Path, logs_dir: Path, stage: str) -> tuple[int, dict]:
    argv = STAGE_ARGV.get(stage)
    if argv is None:
        raise RuntimeError(f"未知阶段: {stage}")

    result = run(argv, rust_root)
    log_path = logs_dir / f"{stage}.log"
    write_command_log(log_path, result)

    if result.returncode == 0:
        return 0, {
            "status": "passed",
            "stage": stage,
            "rust_root": str(rust_root),
            "command": argv,
            "log": str(log_path),
            "summary": [f"Rust {stage} 通过。"],
        }
    return 2, {
        "status": "failed",
        "stage": stage,
        "rust_root": str(rust_root),
        "command": argv,
        "log": str(log_path),
        "summary": summarize_failure(result),
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="在当前 worktree 中受控运行 Rust fmt/check/clippy/build")
    parser.add_argument("--rust-root", help="Rust 工程目录，默认 <repo>/backend")
    parser.add_argument("--task", choices=sorted(ALLOWED_TASKS), default="all")
    args = parser.parse_args()

    cwd = Path.cwd().resolve()
    logs_dir = cwd / "tmp" / "rust-dev" / now_stamp()
    logs_dir.mkdir(parents=True, exist_ok=True)

    try:
        repo = ensure_allowed_worktree(cwd)
        repo_top = Path(repo["top"]).resolve()
        rust_root = ensure_project_dir(repo_top, "backend", "Cargo.toml", args.rust_root)

        stages: list[str]
        if args.task == "all":
            stages = ["fmt", "check", "clippy", "build"]
        else:
            stages = [args.task]

        report: dict[str, object] = {
            "status": "passed",
            "task": args.task,
            "worktree_root": str(repo_top),
            "rust_root": str(rust_root),
            "logs_dir": str(logs_dir),
            "steps": [],
        }

        for stage in stages:
            code, payload = run_stage(rust_root, logs_dir, stage)
            report["steps"].append(payload)
            if payload["status"] == "failed":
                report["status"] = "failed"
                report["summary"] = payload["summary"]
                return finish_report(report, logs_dir)

        report["summary"] = [f"Rust {args.task} 全部通过。"]
        return finish_report(report, logs_dir)

    except RuntimeError as exc:
        return finish_blocked(args.task, str(exc), logs_dir)


if __name__ == "__main__":
    raise SystemExit(main())
